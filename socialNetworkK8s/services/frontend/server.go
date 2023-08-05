package frontend

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"net/url"
	"net/http"
	"net/http/pprof"
	"strconv"
	userpb "socialnetworkk8/services/user/proto"
	composepb "socialnetworkk8/services/compose/proto"
	tlpb "socialnetworkk8/services/timeline/proto"
	homepb "socialnetworkk8/services/home/proto"
	postpb "socialnetworkk8/services/post/proto"
	"socialnetworkk8/services/user"
	"socialnetworkk8/services/compose"
	"socialnetworkk8/services/timeline"
	"socialnetworkk8/services/home"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
	"socialnetworkk8/dialer"
	"socialnetworkk8/registry"
	"socialnetworkk8/tls"
	"github.com/opentracing/opentracing-go"
)

var (
    posttypesMap = map[string]postpb.POST_TYPE {
		"unknown": postpb.POST_TYPE_UNKNOWN,
        "post":    postpb.POST_TYPE_POST,
        "repost":  postpb.POST_TYPE_REPOST,
        "reply":   postpb.POST_TYPE_REPLY,
        "dm":      postpb.POST_TYPE_DM,
    }
)

// Server implements frontend service
type FrontendSrv struct {
	userc     userpb.UserClient
	tlc       tlpb.TimelineClient
	homec     homepb.HomeClient
	composec  composepb.ComposeClient
	IpAddr    string
	Port      int
	record    bool
	Tracer    opentracing.Tracer
	Registry  *registry.Client
	p         *Perf
}

// Run the server
func (s *FrontendSrv) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("Server port must be set")
	}
	log.Info().Msg("Initializing gRPC clients...")
	// user client
	userConn, err := dialer.Dial(user.USER_SRV_NAME, s.Registry.Client, dialer.WithTracer(s.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.userc = userpb.NewUserClient(userConn)
	// timeline client
	tlConn, err := dialer.Dial(timeline.TIMELINE_SRV_NAME, s.Registry.Client, dialer.WithTracer(s.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.tlc = tlpb.NewTimelineClient(tlConn)
	// home client
	homeConn, err := dialer.Dial(home.HOME_SRV_NAME, s.Registry.Client, dialer.WithTracer(s.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.homec = homepb.NewHomeClient(homeConn)
	// compose client
	composeConn, err := dialer.Dial(compose.COMPOSE_SRV_NAME, s.Registry.Client, dialer.WithTracer(s.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.composec = composepb.NewComposeClient(composeConn)
	s.p = MakePerf("social-network-perf/k8s", "social-netowrk")
	s.record = false

	log.Info().Msg("Successfull")

	log.Trace().Msg("frontend before mux")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	//mux := tracing.NewServeMux(s.Tracer)
	mux := http.NewServeMux()
	mux.Handle("/echo", http.HandlerFunc(s.echoHandler))
	mux.Handle("/user", http.HandlerFunc(s.userHandler))
	mux.Handle("/compose", http.HandlerFunc(s.composeHandler))
	mux.Handle("/timeline", http.HandlerFunc(s.timelineHandler))
	mux.Handle("/home", http.HandlerFunc(s.homeHandler))
	mux.Handle("/saveresults", http.HandlerFunc(s.saveResultsHandler))
	mux.Handle("/pprof/cpu", http.HandlerFunc(pprof.Profile))
	mux.Handle("/startrecording", http.HandlerFunc(s.startRecordingHandler))
	log.Trace().Msg("frontend starts serving")

	tlsconfig := tls.GetHttpsOpt()
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: mux,
	}
	if tlsconfig != nil {
		log.Info().Msg("Serving https")
		srv.TLSConfig = tlsconfig
		return srv.ListenAndServeTLS("x509/server_cert.pem", "x509/server_key.pem")
	} else {
		log.Info().Msg("Serving https")
		return srv.ListenAndServe()
	}
}

func (s *FrontendSrv) echoHandler(w http.ResponseWriter, r *http.Request) {
	if s.record {
		defer s.p.TptTick(1.0)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Trace().Msg("starts echoHandler")

	// grab locale from query params or default to en
	msg := r.URL.Query().Get("msg")

	log.Debug().Msg(fmt.Sprintf("echoHandler msg %v", msg))

	res := map[string]interface{}{
		"message": "Echo: " + msg,
	}

	json.NewEncoder(w).Encode(res)
}

func (s *FrontendSrv) userHandler(w http.ResponseWriter, r *http.Request) {
	if s.record {
		defer s.p.TptTick(1.0)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	rawQuery, _ := url.QueryUnescape(r.URL.RawQuery)
	urlQuery, _ := url.ParseQuery(rawQuery)
	log.Debug().Msgf("user request %v\n", rawQuery)

	username, password := urlQuery.Get("username"), urlQuery.Get("password")
	if username == "" || password == "" {
		http.Error(w, "Please specify username and password", http.StatusBadRequest)
		return
	}
	// Check username and password
	res, err := s.userc.Login(
		r.Context(), &userpb.LoginRequest{Username: username,Password: password})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	str := "Login successfully!"
	if res.Ok != user.USER_QUERY_OK {
		str = "Failed. Please check your username and password. "
	}
	reply := map[string]interface{}{"message": str}
	json.NewEncoder(w).Encode(reply)
}

func (s *FrontendSrv) composeHandler(w http.ResponseWriter, r *http.Request) {
	if s.record {
		defer s.p.TptTick(1.0)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	rawQuery, _ := url.QueryUnescape(r.URL.RawQuery)
	urlQuery, _ := url.ParseQuery(rawQuery)
	log.Debug().Msgf("Compose request: %v\n", urlQuery)
	username, useridstr := urlQuery.Get("username"), urlQuery.Get("userid")
	ctx := r.Context()
	var userid int64
	if useridstr == "" {
		if username == "" {
			http.Error(w, "Please specify username or id", http.StatusBadRequest)
			return
		} 
		// retrieve userid
		res, err := s.userc.CheckUser( 
			ctx, &userpb.CheckUserRequest{Usernames: []string{username}})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if res.Ok != user.USER_QUERY_OK {
			http.Error(w, "bad user name or id", http.StatusBadRequest)
			return
		}
		userid = res.Userids[0]
	} else {
		var err error
		userid, err = strconv.ParseInt(useridstr, 10, 64)
		if err != nil {
			http.Error(w, "bad user id format", http.StatusBadRequest)
			return
		}
	}
	// compose a post
	text, posttype, mediastr := urlQuery.Get("text"), urlQuery.Get("posttype"), urlQuery.Get("media")
	mediaids := make([]int64, 0)
	if mediastr != "" {
		for _, idstr := range strings.Split(mediastr, ",") {
			mediaid, err := strconv.ParseInt(idstr, 10, 64)
			if err != nil {
				log.Info().Msgf("Cannot parse media: %v", idstr)
			} else {
				mediaids = append(mediaids, mediaid)
			}
		}
	}
	res, err := s.composec.ComposePost(ctx, &composepb.ComposePostRequest{
		Userid: userid,
		Username: username,
		Text: text,
		Posttype: parsePostTypeString(posttype),
		Mediaids: mediaids,
	})
	if err != nil {
		log.Info().Msgf("Error from compose: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	str := "Compose successfully!"
	if res.Ok != compose.COMPOSE_QUERY_OK {
		str = res.Ok
	}
	reply := map[string]interface{}{"message": str}
	json.NewEncoder(w).Encode(reply)
}

func (s *FrontendSrv) timelineHandler(w http.ResponseWriter, r *http.Request) {
	s.timelineHandlerInner(w, r, false)
}

func (s *FrontendSrv) homeHandler(w http.ResponseWriter, r *http.Request) {
	s.timelineHandlerInner(w, r, true)
}

func (s *FrontendSrv) timelineHandlerInner(w http.ResponseWriter, r *http.Request, isHome bool) {
	if s.record {
		defer s.p.TptTick(1.0)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ctx := r.Context()
	rawQuery, _ := url.QueryUnescape(r.URL.RawQuery)
	urlQuery, _ := url.ParseQuery(rawQuery)
	debugInfo := "Timeline request"
	if isHome {
		debugInfo = "Home timeline request"
	}
	log.Debug().Msgf("%s: %v\n", debugInfo, urlQuery)
	useridstr, startstr, stopstr := 
		urlQuery.Get("userid"), urlQuery.Get("start"),urlQuery.Get("stop")
	var err, err1, err2, err3 error
	var start, stop int64
	userid, err1 := strconv.ParseInt(useridstr, 10, 64)
	if startstr == "" {
		start = 0
	} else {
		start, err2 = strconv.ParseInt(startstr, 10, 32)
	}
	if stopstr == "" {
		stop = 1
	} else {
		stop, err2 = strconv.ParseInt(stopstr, 10, 32)
	}
	if err1 != nil || err2 != nil || err3 != nil {
		http.Error(w, "bad number format in request", http.StatusBadRequest)
		return
	}
	var res = &tlpb.ReadTimelineResponse{}
   	if isHome {
		res, err = s.homec.ReadHomeTimeline(
			ctx, &tlpb.ReadTimelineRequest{Userid: userid, Start: int32(start), Stop: int32(stop)})
	} else {
		res, err = s.tlc.ReadTimeline(
			ctx, &tlpb.ReadTimelineRequest{Userid: userid, Start: int32(start), Stop: int32(stop)})
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	str := "Timeline successfully!"
	postCreators := ""
	postTimes := ""
	postContents := ""
	if res.Ok != timeline.TIMELINE_QUERY_OK {
		str = "Timeline Failed!" + res.Ok
	} else {
		for _, post := range res.Posts {
			postTimes += time.Unix(0, post.Timestamp).Format(time.UnixDate) + "; "
			postCreators += post.Creatoruname + "; "
			postContents += post.Text + "; "
		}
	}
	reply := map[string]interface{}{
		"message": str, "times": postTimes, "contents": postContents, "creators": postCreators}
	json.NewEncoder(w).Encode(reply)
}

func (s *FrontendSrv) startRecordingHandler(w http.ResponseWriter, r *http.Request) {

	s.record = true

	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Debug().Msg("Start recording!")

	str := "Started recording!"

	reply := map[string]interface{}{
		"message": str,
	}

	json.NewEncoder(w).Encode(reply)
}

func (s *FrontendSrv) saveResultsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.p.Done()

	str := "Done!"

	res := map[string]interface{}{
		"message": str,
	}

	json.NewEncoder(w).Encode(res)
}


func parsePostTypeString(str string) (postpb.POST_TYPE) {
    c, ok := posttypesMap[strings.ToLower(str)]
	if !ok {
		c = postpb.POST_TYPE_UNKNOWN
	}
    return c
}
