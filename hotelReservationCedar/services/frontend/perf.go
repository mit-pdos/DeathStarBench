package frontend

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	log "github.com/sirupsen/logrus"
)

// Tracks performance statistics for any cores on which the current process is
// able to run.
type Perf struct {
	mu       sync.Mutex
	fname    string
	done     uint32
	sampleHz int
	tpt      bool
	tpts     []float64
	times    []time.Time
}

// A slight hack for benchmarks which wish to have 2 perf structures (one for
// each realm).
func MakePerf(tptdir string, name string) *Perf {
	p := &Perf{}
	rand.Seed(time.Now().UnixMicro())
	p.fname = tptdir + "/" + name + "-" + strconv.Itoa(rand.Int())
	p.sampleHz = 50
	p.setupTpt()
	return p
}

// Register that an event has happened with a given instantaneous throughput.
func (p *Perf) TptTick(tpt float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tptTickL(tpt)
}

func (p *Perf) tptTickL(tpt float64) {
	// If we aren't recording throughput, return.
	if !p.tpt {
		return
	}

	// If it has been long enough since we started incrementing this slot, seal
	// it and move to the next slot. In this way, we always expect
	// len(p.times) == len(p.tpts) - 1
	if time.Since(p.times[len(p.times)-1]).Milliseconds() > int64(1000/p.sampleHz) {
		p.tpts = append(p.tpts, 0.0)
		p.times = append(p.times, time.Now())
	}

	// Increment the current tpt slot.
	p.tpts[len(p.tpts)-1] += tpt
}

func (p *Perf) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done == 0 {
		atomic.StoreUint32(&p.done, 1)
		p.teardownTpt()
	}
}

func (p *Perf) setupTpt() {
	p.mu.Lock()

	p.tpt = true
	// Pre-allocate a large number of entries (40 secs worth)
	const nsecs = 100
	p.times = make([]time.Time, 0, nsecs*p.sampleHz)
	p.tpts = make([]float64, 0, nsecs*p.sampleHz)

	p.times = append(p.times, time.Now())
	p.tpts = append(p.tpts, 0.0)

	p.mu.Unlock()
}

// Caller holds lock.
func (p *Perf) teardownTpt() {
	if p.tpt {
		p.tpt = false
		s := ""
		// Ignore first entry.
		for i := 0; i < len(p.times); i++ {
			s += fmt.Sprintf("%vus,%f\n", p.times[i].UnixMicro(), p.tpts[i])
		}
		uploadData(p.fname, s)
	}
}

func uploadData(fname string, d string) {
	log.Infof("Writing to output file %v", fname)
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	// Setup the S3 Upload Manager. Also see the SDK doc for the Upload Manager
	// for more information on configuring part size, and concurrency.
	//
	// http://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewUploader
	uploader := s3manager.NewUploader(sess)

	// Upload the file's body to S3 bucket as an object with the key being the
	// same as the filename.
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("9ps3"),

		// Can also use the `filepath` standard library package to modify the
		// filename as need for an S3 object key. Such as turning absolute path
		// to a relative path.
		Key: aws.String(fname),

		// The file to be uploaded. io.ReadSeeker is preferred as the Uploader
		// will be able to optimize memory when uploading large content. io.Reader
		// is supported, but will require buffering of the reader's bytes for
		// each part.
		Body: strings.NewReader(d),
	})
	if err != nil {
		// Print the error and exit.
		log.Fatalf("Unable to upload %q to %q, %v", fname, "s3://9ps3", err)
	}
	log.Infof("Done writing to output file %v", fname)
}
