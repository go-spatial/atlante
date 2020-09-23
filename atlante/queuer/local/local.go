package local

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-spatial/atlante/atlante"
	"github.com/go-spatial/atlante/atlante/queuer"
	"github.com/prometheus/common/log"
)

const (

	// TYPE is the name of the provider
	TYPE = "local"
)

var (
	globalCtx context.Context
)

func init() {
	var cancel context.CancelFunc
	globalCtx, cancel = context.WithCancel(context.Background())
	queuer.Register(TYPE, initFunc, queuer.CleanupFunc(cancel))
}

type jobInfo struct {
	jobid string
	key   string
	job   *atlante.Job
}

func (ji *jobInfo) Reset() {
	ji.jobid = ""
	ji.key = ""
	ji.job = nil
}

type Provider struct {
	atlante     *atlante.Atlante
	jobInfoPool sync.Pool
	jobChannel  chan *jobInfo
	count       *uint32
}

func (p *Provider) jobRunner(ctx context.Context) {
	var (
		err error
		ok  bool
		ji  *jobInfo
	)
	log.Infof("jobRunner started")

	for {
		select {
		case <-ctx.Done():
			log.Infof("jobRunner got context cancel")
			// we need to exit
			return
		case ji, ok = <-p.jobChannel:
			if !ok {
				log.Infof("jobRunner jobChannel closed")
				return
			}
			if ji == nil {
				log.Infof("jobRunner ji is nil")
				continue
			}
			log.Infof("starting job(%v)", ji.jobid)
			_, err = p.atlante.GeneratePDFJob(ctx, *(ji.job), "")
			if err != nil {
				log.Infof("Local runner job(%v) failed: %v", ji.jobid, err)
			}
			p.jobInfoPool.Put(ji)
		}
	}
}

func initFunc(cfg queuer.Config, a *atlante.Atlante) (queuer.Provider, error) {
	runners, _ := cfg.Int("max_runners", nil)
	return NewProvider(globalCtx, a, runners), nil
}

func NewProvider(ctx context.Context, a *atlante.Atlante, runners int) *Provider {
	prv := &Provider{
		atlante: a,
		jobInfoPool: sync.Pool{
			New: func() interface{} { return new(jobInfo) },
		},
		jobChannel: make(chan *jobInfo),
		count:      new(uint32),
	}
	if runners <= 0 {
		runners = 1
	}
	for i := 0; i < runners; i++ {
		go prv.jobRunner(ctx)
	}
	return prv
}

func (p *Provider) Enqueue(key string, job *atlante.Job) (jobid string, err error) {
	if p == nil {
		return "", fmt.Errorf("nil provider")
	}
	if p.jobChannel == nil {
		return "", fmt.Errorf("no queue available")
	}
	idNum := atomic.AddUint32(p.count, 1)
	ji := p.jobInfoPool.Get().(*jobInfo)
	jobid = fmt.Sprintf("%s_%03d", key, idNum)
	ji.Reset()
	ji.jobid = jobid
	ji.key = key
	ji.job = job
	log.Infof("enqueing job(%v)", ji.jobid)
	p.jobChannel <- ji

	return jobid, nil
}
