package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
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

func handlePatch(object interface{}) ([]byte, error) {
	switch object.(type) {
	case *corev1.Pod:
		var pod *corev1.Pod
		pod = object.(*corev1.Pod)
		osNodeSelector, ok := pod.Spec.NodeSelector["beta.kubernetes.io/os"]
		if ok == false {
			glog.Infof("OS node selector is not present, defaulting to windows")
			return []byte(`[{"op":"add","path":"/spec/nodeSelector","value":{"beta.kubernetes.io/os": "windows"}},{"op":"add","path":"/metadata/labels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/runtimeClassName","value":"lcow"}]`), nil
		}

		// check if node selector is set to windows
		runtimeClass := pod.Spec.RuntimeClassName
		if runtimeClass == nil {
			glog.Infof("OS node selector is %v, and runtimeclass is Nil", osNodeSelector)
		} else {
			glog.Infof("OS node selector is %v, and runtimeclass is %v", osNodeSelector, *runtimeClass)
		}

		// if runtime class is not present or it is wcow then set the WCOW specific parameters
		if osNodeSelector == "windows" && (runtimeClass == nil || *runtimeClass == "wcow") {
			return []byte(`[{"op":"add","path":"/metadata/labels","value":{"sandbox-platform": "windows-amd64"}},{"op":"add","path":"/spec/runtimeClassName","value":"wcow"}]`), nil
		}

		// it is possible that this pod is created as part of already muatated deployment/replicaset/statefulset/daemonset
		// then check if runtimeclass is set to lcow. in this case do not apply any patch
		if osNodeSelector == "windows" && *runtimeClass == "lcow" {
			return []byte(`[]`), nil
		}

		// linux
		return []byte(`[{"op":"replace","path":"/spec/nodeSelector/beta.kubernetes.io~1os","value": "windows"},{"op":"add","path":"/metadata/labels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/runtimeClassName","value":"lcow"}]`), nil

	case *appsv1.Deployment:
		var deployment *appsv1.Deployment
		deployment = object.(*appsv1.Deployment)

		osNodeSelector, ok := deployment.Spec.Template.Spec.NodeSelector["beta.kubernetes.io/os"]
		if ok == false {
			glog.Infof("OS node selector is not present, defaulting to windows")
			return []byte(`[{"op":"add","path":"/spec/template/spec/nodeSelector","value":{"beta.kubernetes.io/os": "windows"}},{"op":"add","path":"/spec/template/spec/runtimeClassName","value":"lcow"},{"op":"add","path":"/spec/selector/matchLabels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/template/metadata/labels","value":{"sandbox-platform": "linux-amd64"}}]`), nil
		}

		// check if node selector is set to windows
		runtimeClass := deployment.Spec.Template.Spec.RuntimeClassName
		if runtimeClass == nil {
			glog.Infof("OS node selector is %v, and runtimeclass is Nil", osNodeSelector)
		} else {
			glog.Infof("OS node selector is %v, and runtimeclass is %v", osNodeSelector, *runtimeClass)
		}

		if osNodeSelector == "windows" && (runtimeClass == nil || *runtimeClass == "wcow") {
			return []byte(`[{"op":"add","path":"/spec/selector/matchLabels","value":{"sandbox-platform": "windows-amd64"}},{"op":"add","path":"/spec/template/metadata/labels","value":{"sandbox-platform": "windows-amd64"}},{"op":"add","path":"/spec/template/spec/runtimeClassName","value":"wcow"}]`), nil
		}

		if osNodeSelector == "window" && *runtimeClass == "lcow" {
			return []byte(`[]`), nil
		}

		// linux
		return []byte(`[{"op":"replace","path":"/spec/template/spec/nodeSelector/beta.kubernetes.io~1os","value": "windows"},{"op":"add","path":"/spec/selector/matchLabels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/template/metadata/labels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/template/spec/runtimeClassName","value":"lcow"}]`), nil

	case *appsv1.ReplicaSet:
		var replicaSet *appsv1.ReplicaSet
		replicaSet = object.(*appsv1.ReplicaSet)

		osNodeSelector, ok := replicaSet.Spec.Template.Spec.NodeSelector["beta.kubernetes.io/os"]
		if ok == false {
			glog.Infof("OS node selector is not present, defaulting to windows")
			return []byte(`[{"op":"add","path":"/spec/template/spec/nodeSelector","value":{"beta.kubernetes.io/os": "windows"}},{"op":"add","path":"/spec/template/spec/runtimeClassName","value":"lcow"},{"op":"add","path":"/spec/selector/matchLabels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/template/metadata/labels","value":{"sandbox-platform": "linux-amd64"}}]`), nil
		}

		// check if node selector is set to windows
		runtimeClass := replicaSet.Spec.Template.Spec.RuntimeClassName
		if runtimeClass == nil {
			glog.Infof("OS node selector is %v, and runtimeclass is Nil", osNodeSelector)
		} else {
			glog.Infof("OS node selector is %v, and runtimeclass is %v", osNodeSelector, *runtimeClass)
		}

		if osNodeSelector == "windows" && (runtimeClass == nil || *runtimeClass == "wcow") {
			return []byte(`[{"op":"add","path":"/spec/selector/matchLabels","value":{"sandbox-platform": "windows-amd64"}},{"op":"add","path":"/spec/template/metadata/labels","value":{"sandbox-platform": "windows-amd64"}},{"op":"add","path":"/spec/template/spec/runtimeClassName","value":"wcow"}]`), nil
		}

		if osNodeSelector == "windows" && *runtimeClass == "lcow" {
			return []byte(`[]`), nil
		}

		// linux
		return []byte(`[{"op":"replace","path":"/spec/template/spec/nodeSelector/beta.kubernetes.io~1os","value": "windows"},{"op":"add","path":"/spec/selector/matchLabels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/template/metadata/labels","value":{"sandbox-platform": "linux-amd64"}},{"op":"add","path":"/spec/template/spec/runtimeClassName","value":"lcow"}]`), nil

	}
	return []byte(`[]`), nil
}

func handleValidation(object interface{}) bool {

	switch object.(type) {
	case *corev1.Pod:

		var pod *corev1.Pod
		pod = object.(*corev1.Pod)

		osNodeSelector, ok := pod.Spec.NodeSelector["beta.kubernetes.io/os"]
		if ok == false {
			glog.Infof("OS node selector is not present, Not Allowing")
			return false
		}
		if osNodeSelector != "linux" && osNodeSelector != "windows" {
			glog.Infof("OS node selector is %v, Not Allowing", osNodeSelector)
			return false
		}

		runtimeClass := pod.Spec.RuntimeClassName
		if runtimeClass == nil {
			glog.Infof("Runtime class not present, Not Allowing")
			return false
		}
		if *runtimeClass != "lcow" && *runtimeClass != "wcow" {
			glog.Infof("Runtime class is %v, Not Allowing", *runtimeClass)
			return false
		}

		sandboxlabel, ok := pod.ObjectMeta.Labels["sandbox-platform"]
		if ok == false {
			glog.Infof("Label sandbox-platform is not present, Not Allowing")
			return false
		}
		if sandboxlabel != "linux-amd64" && sandboxlabel != "windows-amd64" {
			glog.Infof("Label sandbox-platform is %v, Not Allowing", sandboxlabel)
			return false
		}

		glog.Infof("All check passed, Allowing")
		return true

	case *appsv1.Deployment:

		var deployment *appsv1.Deployment
		deployment = object.(*appsv1.Deployment)
		osNodeSelector, ok := deployment.Spec.Template.Spec.NodeSelector["beta.kubernetes.io/os"]
		if ok == false {
			glog.Infof("OS node selector is not present, Not Allowing")
			return false
		}
		if osNodeSelector != "linux" && osNodeSelector != "windows" {
			glog.Infof("OS node selector is %v, Not Allowing", osNodeSelector)
			return false
		}

		runtimeClass := deployment.Spec.Template.Spec.RuntimeClassName
		if runtimeClass == nil {
			glog.Infof("Runtime class not present, Not Allowing")
			return false
		}
		if *runtimeClass != "lcow" && *runtimeClass != "wcow" {
			glog.Infof("Runtime class is %v, Not Allowing", *runtimeClass)
			return false
		}

		sandboxlabel, ok := deployment.Spec.Template.ObjectMeta.Labels["sandbox-platform"]
		if ok == false {
			glog.Infof("Label sandbox-platform is not present, Not Allowing")
			return false
		}
		if sandboxlabel != "linux-amd64" && sandboxlabel != "windows-amd64" {
			glog.Infof("Label sandbox-platform is %v, Not Allowing", sandboxlabel)
			return false
		}

		glog.Infof("All check passed, Allowing")
		return true

	case *appsv1.ReplicaSet:
		var replicaSet *appsv1.ReplicaSet
		replicaSet = object.(*appsv1.ReplicaSet)
		osNodeSelector, ok := replicaSet.Spec.Template.Spec.NodeSelector["beta.kubernetes.io/os"]
		if ok == false {
			glog.Infof("OS node selector is not present, Not Allowing")
			return false
		}
		if osNodeSelector != "linux" && osNodeSelector != "windows" {
			glog.Infof("OS node selector is %v, Not Allowing", osNodeSelector)
			return false
		}

		runtimeClass := replicaSet.Spec.Template.Spec.RuntimeClassName
		if runtimeClass == nil {
			glog.Infof("Runtime class not present, Not Allowing")
			return false
		}
		if *runtimeClass != "lcow" && *runtimeClass != "wcow" {
			glog.Infof("Runtime class is %v, Not Allowing", *runtimeClass)
			return false
		}

		sandboxlabel, ok := replicaSet.Spec.Template.ObjectMeta.Labels["sandbox-platform"]
		if ok == false {
			glog.Infof("Label sandbox-platform is not present, Not Allowing")
			return false
		}
		if sandboxlabel != "linux-amd64" && sandboxlabel != "windows-amd64" {
			glog.Infof("Label sandbox-platform is %v, Not Allowing", sandboxlabel)
			return false
		}

		glog.Infof("All check passed, Allowing")
		return true

	}
	return false
}

func unmarshalObject(req *v1beta1.AdmissionRequest) (interface{}, error) {

	glog.Infof("Entering unmarshalObject()")
	var object interface{}

	switch req.Kind.Kind {
	case "Pod":
		var pod corev1.Pod
		if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return object, err
		}
		object = &pod
		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v", req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return object, err
		}
		object = &deployment
		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v", req.Kind, req.Namespace, req.Name, deployment.Name, req.UID, req.Operation, req.UserInfo)

	case "ReplicaSet":
		var replicaSet appsv1.ReplicaSet
		if err := json.Unmarshal(req.Object.Raw, &replicaSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return object, err
		}
		object = &replicaSet
		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v", req.Kind, req.Namespace, req.Name, replicaSet.Name, req.UID, req.Operation, req.UserInfo)

	case "StatefulSet":
		var stateFulSet appsv1.StatefulSet
		if err := json.Unmarshal(req.Object.Raw, &stateFulSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return object, err
		}
		object = &stateFulSet
		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v", req.Kind, req.Namespace, req.Name, stateFulSet.Name, req.UID, req.Operation, req.UserInfo)
	}

	return object, nil
}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.Infof("Entering mutate()")
	req := ar.Request
	/*
		// check if this request is user created or system created as part of deployment or replicaset
		if strings.Contains(req.UserInfo.Username, "serviceaccount") == true {
			glog.Infof("Not Mutating AdmissionReview for Kind=%v, Namespace=%v (%v) UID=%v patchOperation=%v UserInfo=%v", req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}
	*/

	if object, err := unmarshalObject(req); err == nil {
		switch object.(type) {
		default:
			// TODO : test this negative case
			// If User has configured the webhook for not implemented object then don't apply any patch
			reviewResponse := v1beta1.AdmissionResponse{}
			reviewResponse.Allowed = true
			reviewResponse.Patch = []byte(`[]`)
			pt := v1beta1.PatchTypeJSONPatch
			reviewResponse.PatchType = &pt

			return &reviewResponse

		case *corev1.Pod:
			var pod *corev1.Pod
			pod = object.(*corev1.Pod)
			patchBytes, err := handlePatch(pod)
			glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
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
		case *appsv1.Deployment:
			var deployment *appsv1.Deployment
			deployment = object.(*appsv1.Deployment)
			patchBytes, err := handlePatch(deployment)
			glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
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

		case *appsv1.ReplicaSet:
			var replicaSet *appsv1.ReplicaSet
			replicaSet = object.(*appsv1.ReplicaSet)
			patchBytes, err := handlePatch(replicaSet)
			glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
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

		case *appsv1.StatefulSet:
			var stateFulSet *appsv1.StatefulSet
			stateFulSet = object.(*appsv1.StatefulSet)
			patchBytes, err := handlePatch(stateFulSet)
			glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
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
	} else {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: "tmp",
		},
	}

}

// pod validation
func (whsvr *WebhookServer) validate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.Infof("Entering validate()")
	req := ar.Request

	/*
		// check if this request is user created or system created as part of deployment or replicaset
		if strings.Contains(req.UserInfo.Username, "serviceaccount") == true {
			glog.Infof("Not Validating AdmissionReview for Kind=%v, Namespace=%v (%v) UID=%v patchOperation=%v UserInfo=%v", req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}
	*/

	if object, err := unmarshalObject(req); err == nil {
		switch object.(type) {
		default:
			// TODO : test this negative case
			// If User has configured the webhook for not implemented object then don't apply any patch
			reviewResponse := v1beta1.AdmissionResponse{}
			reviewResponse.Allowed = true
			return &reviewResponse

		case *corev1.Pod:
			var pod *corev1.Pod
			pod = object.(*corev1.Pod)
			allowed := handleValidation(pod)
			var message string
			if allowed == true {
				message = "Allowed"
			} else {
				message = "Not Allowed"
			}

			return &v1beta1.AdmissionResponse{
				Allowed: allowed,
				Result: &metav1.Status{
					Message: message,
				},
			}
		case *appsv1.Deployment:
			var deployment *appsv1.Deployment
			deployment = object.(*appsv1.Deployment)
			allowed := handleValidation(deployment)
			var message string
			if allowed == true {
				message = "Allowed"
			} else {
				message = "Not Allowed"
			}

			return &v1beta1.AdmissionResponse{
				Allowed: allowed,
				Result: &metav1.Status{
					Message: message,
				},
			}
		}
	}
	return &v1beta1.AdmissionResponse{
		Allowed: false,
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) mutateRequest(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Entering mutateRequest()")
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

// Serve method for webhook server
func (whsvr *WebhookServer) validateRequest(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Entering validateRequest()")
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
		admissionResponse = whsvr.validate(&ar)
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
