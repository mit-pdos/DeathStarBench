package socialnetworkk8_test

import (
	"crypto/sha256"
	"strconv"
	"fmt"
	"github.com/rs/zerolog/log"
	"gopkg.in/mgo.v2"
	"socialnetworkk8/services/cacheclnt"
	"socialnetworkk8/services/user"
	"socialnetworkk8/tune"
	"os/exec"
	"time"
)

const (
	NUSER = 10
	MONGO_FWD_PORT = "9090"
)

var tu *TestUtil

func StartFowarding(service, testPort, targetPort string) (*exec.Cmd, error) {
	cmd := exec.Command("kubectl", "port-forward", "svc/"+service, testPort+":"+targetPort)
	if err := cmd.Start(); err != nil {
		return nil,  err
	}
	time.Sleep(1*time.Second)
	return cmd, nil
}

type TestUtil struct {
	mongoSess    *mgo.Session
	cachec       *cacheclnt.CacheClnt
	fcmd         *exec.Cmd
}

func makeTestUtil() (*TestUtil, error) {
	tune.Init()
	log.Info().Msg("Start cache and DB connections")
	cachec := cacheclnt.MakeCacheClnt() 
	fcmd, err := StartFowarding("mongodb-sn", MONGO_FWD_PORT, "27017")
	if err != nil {
		log.Error().Msgf("Cannot forward mongodb port: %v", err)
		return nil, err
	}
	session, err := mgo.Dial("localhost:"+MONGO_FWD_PORT)
	if err != nil {
		log.Error().Msgf("Cannot dial to Mongo: %v", err)
		return nil, err
	}
	log.Info().Msg("New session successfull...")
	return &TestUtil{session, cachec, fcmd}, nil
}

func (tu *TestUtil) clearDB() error {
	if err := tu.mongoSess.DB("socialnetwork").C("user").DropCollection(); err != nil {
		log.Error().Msgf("Cannot dop user collection: %v", err)
		return err
	}
	return nil
}

/*
func (tu *TestUtil) ClearCache() {

}
*/

func (tu *TestUtil) initUsers() error {
	// create NUSER test users
	for i := 0; i < NUSER; i++ {
		suffix := strconv.Itoa(i)
		newUser := user.User{
			Userid: int64(i), 
			Username: "user_" + suffix, 
			Lastname: "Lastname" + suffix, 
			Firstname: "Firstname" + suffix, 
			Password: fmt.Sprintf("%x", sha256.Sum256([]byte("p_user_" + suffix)))}
		if err := tu.mongoSess.DB("socialnetwork").C("user").Insert(&newUser); err != nil {
			log.Fatal().Msg(err.Error())
			return err
		}
	}
	return nil
}

func (tu *TestUtil) Close() {
	tu.mongoSess.Close()
	tu.fcmd.Process.Kill()
} 

func init() {
	tu, _ = makeTestUtil()
	defer tu.Close()
	tu.clearDB()
	tu.initUsers()
}