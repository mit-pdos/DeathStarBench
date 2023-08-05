package text

import (
	"regexp"
	"fmt"
	"sync"
	"net"
	"github.com/google/uuid"
	"socialnetworkk8/registry"
	"socialnetworkk8/tune"
	"socialnetworkk8/tls"
	"socialnetworkk8/dialer"
	"socialnetworkk8/services/user"
	"socialnetworkk8/services/url"
	"socialnetworkk8/services/text/proto"
	userpb "socialnetworkk8/services/user/proto"
	urlpb "socialnetworkk8/services/url/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	TEXT_SRV_NAME = "srv-text"
	TEXT_QUERY_OK = "OK"
)

var mentionRegex = regexp.MustCompile("@[a-zA-Z0-9-_]+") 
var urlRegex = regexp.MustCompile("(http://|https://)([a-zA-Z0-9_!~*'().&=+$%-/]+)")

type TextSrv struct {
	proto.UnimplementedTextServer 
	uuid         string
	Registry     *registry.Client
	userc        userpb.UserClient
	urlc         urlpb.UrlClient
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
}

func MakeTextSrv() *TextSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, _ := registry.RegisterByConfig("Text")
	return &TextSrv{Port:serv_port, IpAddr:serv_ip, Tracer:tracer, Registry:registry}
}

// Run starts the server
func (tsrv *TextSrv) Run() error {
	if tsrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("Initializing gRPC clients...")
	userConn, err := dialer.Dial(user.USER_SRV_NAME, tsrv.Registry.Client, dialer.WithTracer(tsrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	tsrv.userc = userpb.NewUserClient(userConn)
	urlConn, err := dialer.Dial(url.URL_SRV_NAME, tsrv.Registry.Client, dialer.WithTracer(tsrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	tsrv.urlc = urlpb.NewUrlClient(urlConn)
	tsrv.uuid = uuid.New().String()
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterTextServer(grpcSrv, tsrv)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", tsrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	err = tsrv.Registry.Register(TEXT_SRV_NAME, tsrv.uuid, tsrv.IpAddr, tsrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}

func (tsrv *TextSrv) ProcessText(
		ctx context.Context, req *proto.ProcessTextRequest) (*proto.ProcessTextResponse, error) {
	res := &proto.ProcessTextResponse{}
	res.Ok = "No. "
	if req.Text == "" {
		res.Ok = "Cannot process empty text." 
		return res, nil
	}
	// find mentions and urls
	mentions := mentionRegex.FindAllString(req.Text, -1)
	mentionsL := len(mentions)
	usernames := make([]string, mentionsL)
	for idx, mention := range mentions {
		usernames[idx] = mention[1:]
	}
	userArg := &userpb.CheckUserRequest{Usernames: usernames}
	userRes := &userpb.CheckUserResponse{}

	urlIndices := urlRegex.FindAllStringIndex(req.Text, -1)
	urlIndicesL := len(urlIndices)
	extendedUrls := make([]string, urlIndicesL)
	for idx, loc := range urlIndices {
		extendedUrls[idx] = req.Text[loc[0]:loc[1]]
	}
	urlArg := &urlpb.ComposeUrlsRequest{Extendedurls: extendedUrls}
	urlRes := &urlpb.ComposeUrlsResponse{}

	// concurrent RPC calls
	var wg sync.WaitGroup
	var userErr, urlErr error
	if mentionsL > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userRes, userErr = tsrv.userc.CheckUser(ctx, userArg)
		}()
	}
	if urlIndicesL > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			urlRes, urlErr = tsrv.urlc.ComposeUrls(ctx, urlArg)
		}()
	}
	wg.Wait()
	res.Text = req.Text
	if userErr != nil || urlErr != nil {
		return nil, fmt.Errorf("%w; %w", userErr, urlErr)
	} 

	// process mentions
	for idx, userid := range userRes.Userids {
		if userid >= 0 {
			res.Usermentions = append(res.Usermentions, userid)
		} else {
			log.Warn().Msgf("User %v does not exist!", usernames[idx])
		}
	}

	// process urls and text
	if urlIndicesL > 0 { 
		if urlRes.Ok != url.URL_QUERY_OK {
			log.Warn().Msgf("cannot process urls %v!", extendedUrls)
			res.Ok += urlRes.Ok
			return res, nil
		} else {
			res.Urls = urlRes.Shorturls
			res.Text = ""
			prevLoc := 0
			for idx, loc := range urlIndices {
				res.Text += req.Text[prevLoc : loc[0]] + urlRes.Shorturls[idx]
				prevLoc = loc[1]
			}
			res.Text += req.Text[urlIndices[urlIndicesL-1][1]:]
		}
	}
	res.Ok = TEXT_QUERY_OK
	return res, nil
} 
