/*******************************************************************************
 * Copyright 2019 Samsung Electronics All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 *******************************************************************************/

/*
 * Edge Orchestration
 *
 * Edge Orchestration support to deliver distributed service process environment.
 *
 * API version: v1-20190307
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

// Package main provides C interface for orchestration
package main

///*******************************************************************************
// * Copyright 2019 Samsung Electronics All Rights Reserved.
// *
// * Licensed under the Apache License, Version 2.0 (the "License");
// * you may not use this file except in compliance with the License.
// * You may obtain a copy of the License at
// *
// * http://www.apache.org/licenses/LICENSE-2.0
// *
// * Unless required by applicable law or agreed to in writing, software
// * distributed under the License is distributed on an "AS IS" BASIS,
// * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// * See the License for the specific language governing permissions and
// * limitations under the License.
// *
// *******************************************************************************/
//struct RequestServiceInfo {
//	char* ExecutionType;
//	char* ExeCmd;
//};
//struct TargetInfo {
//	char* ExecutionType;
//	char* Target;
//};
//struct ResponseService {
//	char*      Message;
//	char*      ServiceName;
//	struct TargetInfo RemoteTargetInfo;
//};
import "C"
import (
	"flag"
	"log"
	"strings"
	"sync"

	"common/logmgr"

	configuremgr "controller/configuremgr/native"
	"controller/discoverymgr"
	scoringmgr "controller/scoringmgr"
	"controller/servicemgr"
	"controller/servicemgr/executor/nativeexecutor"

	"orchestrationapi"

	"restinterface/cipher/sha256"
	"restinterface/client/restclient"
	"restinterface/internalhandler"
	"restinterface/route"
)

const logPrefix = "interface"

// Handle Platform Dependencies
const (
	platform      = "linux"
	executionType = "rpm"

	logPath = "/var/log/edge-orchestration"
	edgeDir = "/etc/edge-orchestration/"

	configPath = edgeDir + "apps"

	cipherKeyFilePath = edgeDir + "orchestration_userID.txt"
	deviceIDFilePath  = edgeDir + "orchestration_deviceID.txt"
)

var (
	flagVersion                  bool
	commitID, version, buildTime string

	orcheEngine orchestrationapi.Orche
)

//export OrchestrationInit
func OrchestrationInit() (errCode C.int) {
	flag.BoolVar(&flagVersion, "v", false, "if true, print version and exit")
	flag.BoolVar(&flagVersion, "version", false, "if true, print version and exit")
	flag.Parse()

	logmgr.Init(logPath)
	log.Printf("[%s] OrchestrationInit", logPrefix)
	log.Println(">>> commitID  : ", commitID)
	log.Println(">>> version   : ", version)
	log.Println(">>> buildTime : ", buildTime)

	restIns := restclient.GetRestClient()
	restIns.SetCipher(sha256.GetCipher(cipherKeyFilePath))

	servicemgr.GetInstance().SetClient(restIns)

	builder := orchestrationapi.OrchestrationBuilder{}
	builder.SetWatcher(configuremgr.GetInstance(configPath))
	builder.SetDiscovery(discoverymgr.GetInstance())
	builder.SetScoring(scoringmgr.GetInstance())
	builder.SetService(servicemgr.GetInstance())
	builder.SetExecutor(nativeexecutor.GetInstance())
	builder.SetClient(restIns)
	orcheEngine = builder.Build()
	if orcheEngine == nil {
		log.Fatalf("[%s] Orchestaration initalize fail", logPrefix)
		return
	}

	orcheEngine.Start(deviceIDFilePath, platform, executionType)

	restEdgeRouter := route.NewRestRouter()

	internalapi, err := orchestrationapi.GetInternalAPI()
	if err != nil {
		log.Fatalf("[%s] Orchestaration internal api : %s", logPrefix, err.Error())
	}
	ihandle := internalhandler.GetHandler()
	ihandle.SetOrchestrationAPI(internalapi)
	ihandle.SetCipher(sha256.GetCipher(cipherKeyFilePath))
	restEdgeRouter.Add(ihandle)
	restEdgeRouter.Start()

	errCode = 0
	log.Println(logPrefix, "orchestration init done")

	return
}

//export OrchestrationRequestService
func OrchestrationRequestService(cAppName *C.char, serviceInfo *C.struct_RequestServiceInfo, count C.int) C.struct_ResponseService {
	log.Printf("[%s] OrchestrationRequestService", logPrefix)

	appName := C.GoString(cAppName)

	requestInfos := make([]orchestrationapi.RequestServiceInfo, count)
	for _, requestInfo := range requestInfos {
		requestInfo.ExecutionType = C.GoString(serviceInfo.ExecutionType)
		args := strings.Split(C.GoString(serviceInfo.ExeCmd), " ")
		if strings.Compare(args[0], "") == 0 {
			args = nil
		}
		copy(requestInfo.ExeCmd, args)
	}

	log.Println("appName:", appName, "infos:", requestInfos)
	externalAPI, err := orchestrationapi.GetExternalAPI()
	if err != nil {
		log.Fatalf("[%s] Orchestaration external api : %s", logPrefix, err.Error())
	}

	res := externalAPI.RequestService(orchestrationapi.ReqeustService{ServiceName: appName, ServiceInfo: requestInfos})
	log.Println("requestService handle : ", res)

	ret := C.struct_ResponseService{}
	ret.Message = C.CString(res.Message)
	ret.ServiceName = C.CString(res.ServiceName)
	ret.RemoteTargetInfo.ExecutionType = C.CString(res.RemoteTargetInfo.ExecutionType)
	ret.RemoteTargetInfo.Target = C.CString(res.RemoteTargetInfo.Target)

	return ret
}

var count int
var mtx sync.Mutex

//export PrintLog
func PrintLog(cMsg *C.char) (count C.int) {
	mtx.Lock()
	msg := C.GoString(cMsg)
	defer mtx.Unlock()
	log.Printf(msg)
	count++
	return
}

func main() {

}