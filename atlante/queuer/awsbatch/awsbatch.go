package awsbatch

import (
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/go-spatial/maptoolkit/atlante"
	cfgaws "github.com/go-spatial/maptoolkit/atlante/config/aws"
	"github.com/go-spatial/maptoolkit/atlante/internal/env"
	"github.com/go-spatial/maptoolkit/atlante/queuer"
	"github.com/prometheus/common/log"
)

const (

	// TYPE is the name of the provider
	TYPE = "awsbatch"

	// ConfigKeyJobDefinition is the key in the config for the job definition
	ConfigKeyJobDefinition = "job_definition"

	// ConfigKeyJobName is the key in the config
	ConfigKeyJobName = "job_name"

	// ConfigKeyJobQueue is the key in the config
	ConfigKeyJobQueue = "job_queue"

	// ConfigKeyJobParams is the key in the config
	ConfigKeyJobParams = "job_parameters"

	// ConfigKeyJobObjectKey is the config key for the job object that will be used
	// added to the params
	ConfigKeyJobObjectKey = "job_object_key"

	// DefaultJobObjectKey is the default that is used by if job_object_key
	// is not set.
	DefaultJobObjectKey = "job-data"
)

func init() {
	queuer.Register(TYPE, initFunc, nil)
}

// Provider implements the queuer interface
type Provider struct {
	Definition string
	Name       string
	Queue      string
	Params     map[string]string
	ObjectKey  string
	Client     *batch.Batch
}

func initFunc(cfg queuer.Config) (queuer.Provider, error) {

	var emptyStr = ""
	var queue Provider

	sess, err := cfgaws.NewSession(cfg)
	if err != nil {
		return nil, err
	}
	queue.Client = batch.New(sess)

	queue.Definition, _ = cfg.String(ConfigKeyJobDefinition, nil)
	queue.Name, err = cfg.String(ConfigKeyJobName, &emptyStr)
	if err != nil {
		return nil, err
	}
	queue.Queue, err = cfg.String(ConfigKeyJobQueue, nil)
	if err != nil {
		return nil, err
	}
	queue.Params = make(map[string]string)
	jobParamMap, _ := cfg.Map(ConfigKeyJobParams)
	// Let's convert this to a env.Dict so that we can
	// get the keys.
	// TODO(gdey): Introduce a Keys() to dict.Dicter interface
	// to get the keys that it has. This way I don't need to do this
	// Type Assertion
	jobParamEnvDict, ok := jobParamMap.(env.Dict)

	if ok {
		for key, val := range jobParamEnvDict {
			queue.Params[key] = fmt.Sprintf("%v", val)
		}
	}

	queue.ObjectKey, _ = cfg.String(ConfigKeyJobObjectKey, nil)

	if queue.ObjectKey == "" {
		queue.ObjectKey = DefaultJobObjectKey
	}

	delete(queue.Params, queue.ObjectKey)

	return &queue, nil
}

var (
	nameCleanRx1 = regexp.MustCompile(`[^[:alnum:]_-]`)
	nameCleanRx2 = regexp.MustCompile(`_+`)
)

func truncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	var numRunes = 0
	for index := range str {
		numRunes++
		if numRunes > length {
			return str[:index]
		}
	}
	return str
}

// Enqueue submits  the given job to aws batch
func (p *Provider) Enqueue(key string, job *atlante.Job) (jobid string, err error) {
	var params = make(map[string]*string)
	for key := range p.Params {
		pstr := p.Params[key]
		params[key] = &pstr
	}
	jobstr, err := job.Base64Marshal()
	if err != nil {
		return "", err
	}

	name := p.Name
	if name == "" {
		name = key
	}

	// Clean up name:
	// The name of the job. The first character must be alphanumeric, and up to
	// 128 letters (uppercase and lowercase), numbers, hyphens, and underscores
	// are allowed.
	//
	// JobName is a required field
	name = truncateString(
		nameCleanRx2.ReplaceAllString(
			nameCleanRx1.ReplaceAllString(name, "_"),
			"_",
		),
		128,
	)

	//first change anything not a letter,number,hypherm or underscore to an underscore.

	params[p.ObjectKey] = &jobstr
	input := &batch.SubmitJobInput{
		JobDefinition: aws.String(p.Definition),
		JobName:       aws.String(name),
		JobQueue:      aws.String(p.Queue),
		Parameters:    params,
	}
	result, err := p.Client.SubmitJob(input)
	if err != nil {
		log.Warnf("Got the error submitting job: %v", err)
		return "", err
	}
	if result.JobId != nil {
		return *result.JobId, nil
	}
	return "", nil
	/* TODO(gdey): should we wrap our error and handle the
	   different cases better?
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case batch.ErrCodeClientException:
				fmt.Println(batch.ErrCodeClientException, aerr.Error())
			case batch.ErrCodeServerException:
				fmt.Println(batch.ErrCodeServerException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	*/
}
