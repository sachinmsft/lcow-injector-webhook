package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

type WebhookServer struct {
	server *http.Server
}

// Webhook Server parameters
type WhSvrParameters struct {
	port     int    // webhook server port
	certFile string // path to the x509 certificate for https
	keyFile  string // path to the x509 private key matching `CertFile`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

const (
	lcowRuntimeClassPatch string = `[
		 {"op":"add","path":"/spec/RuntimeClassName","value":"lcow"}
	]`

	wcowRuntimeClassPatch string = `[
		 {"op":"add","path":"/spec/RuntimeClassName","value":"wcow"}
	]`

	lcowSandboxPlatformPatch string = `[
		{"op":"add","path":"/spec/metadata/Labels","value":[{"sandbox-platform":"linux-amd64"}]}
	]`

	wcowSandboxPlatformPatch string = `[
		 {"op":"add","path":"/spec/metadata/Labels","value":[{"sandbox-platform":"windows-amd64"}]}
	]`
)

func handlePodPatch(pod *corev1.Pod) ([]byte, error) {

	var patch string
	// check if node selector is set to linux
	if pod.Spec.NodeSelector["beta.kubernetes.io/os"] == "linux" {
		glog.Infof("Node selector is linux")
		// remove the linux node selector and add windows so that pod should schedule on windows node
		//TODO : remove linux and add windows node selector

		patch = lcowSandboxPlatformPatch
	} else {

	}
	return []byte(patch), nil
}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request

	switch req.Kind.Kind {
	case "Pod":
		var pod corev1.Pod
		if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
			req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

		patchBytes, err := handlePodPatch(&pod)
		if err != nil {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		} else {
			reviewResponse := v1beta1.AdmissionResponse{}
			reviewResponse.Allowed = true
			reviewResponse.Patch = patchBytes
			pt := v1beta1.PatchTypeJSONPatch
			reviewResponse.PatchType = &pt

			return &reviewResponse
		}

	}

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: "tmp",
		},
	}
	// apply logic

}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = whsvr.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
