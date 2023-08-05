package compose

import (
	"time"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"github.com/google/uuid"
	"socialnetworkk8/registry"
	"socialnetworkk8/tune"
	"socialnetworkk8/services/compose/proto"
	"socialnetworkk8/services/text"
	textpb "socialnetworkk8/services/text/proto"
	"socialnetworkk8/services/post"
	postpb "socialnetworkk8/services/post/proto"
	"socialnetworkk8/services/timeline"
	tlpb "socialnetworkk8/services/timeline/proto"
	"socialnetworkk8/services/home"
	homepb "socialnetworkk8/services/home/proto"
	"socialnetworkk8/tls"
	"socialnetworkk8/dialer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	COMPOSE_SRV_NAME = "srv-compose"
	COMPOSE_QUERY_OK = "OK"
)

// Server implements the compose service
type ComposeSrv struct {
	proto.UnimplementedComposeServer
	uuid   		 string
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	textc        textpb.TextClient
	postc        postpb.PostStorageClient
	tlc          tlpb.TimelineClient
	homec        homepb.HomeClient
	Port         int
	IpAddr       string
	sid          int32 // sid is a random number between 0 and 2^30
	pcount       int32 //This server may overflow with over 2^31 composes
    mu           sync.Mutex
}

func MakeComposeSrv() *ComposeSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, _ := registry.RegisterByConfig("Compose")
	return &ComposeSrv{Port: serv_port, IpAddr: serv_ip, Tracer: tracer, Registry: registry}
}

// Run starts the server
func (csrv *ComposeSrv) Run() error {
	if csrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	textConn, err := dialer.Dial(text.TEXT_SRV_NAME, csrv.Registry.Client, dialer.WithTracer(csrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	csrv.textc = textpb.NewTextClient(textConn)
	postConn, err := dialer.Dial(post.POST_SRV_NAME, csrv.Registry.Client, dialer.WithTracer(csrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	csrv.postc = postpb.NewPostStorageClient(postConn)

	tlConn, err := dialer.Dial(timeline.TIMELINE_SRV_NAME, csrv.Registry.Client, dialer.WithTracer(csrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	csrv.tlc = tlpb.NewTimelineClient(tlConn)
	homeConn, err := dialer.Dial(home.HOME_SRV_NAME, csrv.Registry.Client, dialer.WithTracer(csrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	csrv.homec = homepb.NewHomeClient(homeConn)
	csrv.uuid = uuid.New().String()
	csrv.sid = rand.Int31n(536870912) // 2^29
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)

	proto.RegisterComposeServer(grpcSrv, csrv)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", csrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	err = csrv.Registry.Register(COMPOSE_SRV_NAME, csrv.uuid, csrv.IpAddr, csrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}

// Shutdown cleans up any processes
func (csrv *ComposeSrv) Shutdown() {
	csrv.Registry.Deregister(csrv.uuid)
}

func (csrv *ComposeSrv) ComposePost(
		ctx context.Context, req *proto.ComposePostRequest) (*proto.ComposePostResponse, error) {
	log.Debug().Msgf("Recieved compose request: %v", req)
	res := &proto.ComposePostResponse{Ok: "No"}
	timestamp := time.Now().UnixNano()
	if req.Text == "" {
		res.Ok = "Cannot compose empty post!"
		return res, nil
	}
	// process text
	textReq := &textpb.ProcessTextRequest{Text: req.Text}
	textRes, err := csrv.textc.ProcessText(ctx, textReq)
	if err != nil {
		log.Error().Msgf("Error processing text: %v")
		return res, err
	}
	if textRes.Ok != text.TEXT_QUERY_OK {
		res.Ok += " Text Error: " + textRes.Ok
		return res, nil
	} 
	// create post
	newPost := &postpb.Post{
		Postid: csrv.getNextPostId(),
		Posttype: req.Posttype,
		Timestamp: timestamp,
		Creator: req.Userid,
		Creatoruname: req.Username,
		Text: textRes.Text,
		Usermentions: textRes.Usermentions,
		Urls: textRes.Urls,
		Medias: req.Mediaids,
	}
	log.Debug().Msgf("composing post: %v", newPost)
	
	// concurrently add post to storage and timelines
	var wg sync.WaitGroup
	var postErr, tlErr, homeErr error
	postReq := &postpb.StorePostRequest{Post: newPost}
	postRes := &postpb.StorePostResponse{}
	tlReq := &tlpb.WriteTimelineRequest{
		Userid: req.Userid, 
		Postid: newPost.Postid, 
		Timestamp: newPost.Timestamp}
	tlRes := &tlpb.WriteTimelineResponse{}
	homeReq := &homepb.WriteHomeTimelineRequest{
		Usermentionids: newPost.Usermentions, 
		Userid: req.Userid, 
		Postid: newPost.Postid, 
		Timestamp: newPost.Timestamp}
	homeRes := &tlpb.WriteTimelineResponse{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		postRes, postErr = csrv.postc.StorePost(ctx, postReq)
	}()
	go func() {
		defer wg.Done()
		tlRes, tlErr = csrv.tlc.WriteTimeline(ctx, tlReq) 
	}()
	go func() {
		defer wg.Done()
		homeRes, homeErr = csrv.homec.WriteHomeTimeline(ctx, homeReq)
	}()
	wg.Wait()
	if postErr != nil || tlErr != nil || homeErr != nil {
		return nil, fmt.Errorf("%w; %w; %w", postErr, tlErr, homeErr)
	}
	if postRes.Ok != post.POST_QUERY_OK {
		res.Ok += " Post Error: " + postRes.Ok
		return res, nil
	} 
	if tlRes.Ok != timeline.TIMELINE_QUERY_OK {
		res.Ok += " Timeline Error: " + tlRes.Ok
		return res, nil
	}
	if homeRes.Ok != home.HOME_QUERY_OK {
		res.Ok += " Home Error: " + homeRes.Ok
		return res, nil
	}
	res.Ok = COMPOSE_QUERY_OK
	return res, nil
}

func (csrv *ComposeSrv) incCountSafe() int32 {
	csrv.mu.Lock()
	defer csrv.mu.Unlock()
	csrv.pcount++
	return csrv.pcount
}

func (csrv *ComposeSrv) getNextPostId() int64 {
	return int64(csrv.sid)*1e10 + int64(csrv.incCountSafe())
}
