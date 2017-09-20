package edge

import (
	"encoding/json"
	"fmt"
	"github.com/streamsets/datacollector-edge/container/common"
	"github.com/streamsets/datacollector-edge/container/dpm"
	"github.com/streamsets/datacollector-edge/container/execution/manager"
	"github.com/streamsets/datacollector-edge/container/http"
	"github.com/streamsets/datacollector-edge/container/store"
	"github.com/streamsets/datacollector-edge/container/util"
	"log"
	"os"
)

const (
	DefaultLogFilePath    = "/log/edge.log"
	DefaultConfigFilePath = "/etc/edge.conf"
	DEBUG                 = "DEBUG"
	WARN                  = "WARN"
	ERROR                 = "ERROR"
	INFO                  = "INFO"
)

type DataCollectorEdgeMain struct {
	Config            *Config
	BuildInfo         *common.BuildInfo
	RuntimeInfo       *common.RuntimeInfo
	WebServerTask     *http.WebServerTask
	Manager           *manager.PipelineManager
	PipelineStoreTask store.PipelineStoreTask
}

func DoMain(
	baseDir string,
	debugFlag bool,
	startFlag string,
	runtimeParametersFlag string,
) (*DataCollectorEdgeMain, error) {
	dataCollectorEdge, _ := newDataCollectorEdge(baseDir, debugFlag)

	if len(startFlag) > 0 {
		var runtimeParameters map[string]interface{}
		if len(runtimeParametersFlag) > 0 {
			err := json.Unmarshal([]byte(runtimeParametersFlag), &runtimeParameters)
			if err != nil {
				panic(err)
			}
		}

		fmt.Println("Starting Pipeline: ", startFlag)
		state, err := dataCollectorEdge.Manager.GetRunner(startFlag).GetStatus()
		if state != nil && state.Status == common.RUNNING {
			// If status is running, change it back to stopped
			dataCollectorEdge.Manager.StopPipeline(startFlag)
		}

		state, err = dataCollectorEdge.Manager.StartPipeline(startFlag, runtimeParameters)
		if err != nil {
			panic(err)
		}
		stateJson, _ := json.Marshal(state)
		fmt.Println(string(stateJson))
	}

	return dataCollectorEdge, nil
}

func newDataCollectorEdge(baseDir string, debugFlag bool) (*DataCollectorEdgeMain, error) {
	initializeLog(debugFlag, baseDir)
	log.Println("[INFO] Base Dir: ", baseDir)

	config := NewConfig()
	config.FromTomlFile(baseDir + DefaultConfigFilePath)

	hostName, _ := os.Hostname()
	var httpUrl = "http://" + hostName + config.Http.BindAddress

	buildInfo, _ := common.NewBuildInfo()
	runtimeInfo, _ := common.NewRuntimeInfo(httpUrl, baseDir)
	pipelineStoreTask := store.NewFilePipelineStoreTask(*runtimeInfo)
	pipelineManager, _ := manager.NewManager(config.Execution, *runtimeInfo, pipelineStoreTask)
	webServerTask, _ := http.NewWebServerTask(config.Http, buildInfo, pipelineManager, pipelineStoreTask)
	dpm.RegisterWithDPM(config.DPM, buildInfo, runtimeInfo)

	return &DataCollectorEdgeMain{
		Config:            config,
		BuildInfo:         buildInfo,
		RuntimeInfo:       runtimeInfo,
		WebServerTask:     webServerTask,
		Manager:           pipelineManager,
		PipelineStoreTask: pipelineStoreTask,
	}, nil
}

func initializeLog(debugFlag bool, baseDir string) {
	minLevel := util.LogLevel(WARN)
	if debugFlag {
		minLevel = util.LogLevel(DEBUG)
	}

	loggerFile, _ := os.OpenFile(baseDir+DefaultLogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	logFilter := &util.LevelFilter{
		Levels:   []util.LogLevel{DEBUG, WARN, ERROR, INFO},
		MinLevel: minLevel,
		Writer:   loggerFile,
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(logFilter)
}