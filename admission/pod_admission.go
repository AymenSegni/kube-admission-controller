package admission

import (
	"encoding/json"
	"net/http"

	"github.com/aymensegni/kube-admission-controller/rules"
	"github.com/labstack/echo"
	v1adm "k8s.io/api/admission/v1"
	v1pod "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AdmitPods(denyLatestTag bool, registryWhiteList []string) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Logger().Infof("Received admission request")
		var admissionReview v1adm.AdmissionReview

		err := c.Bind(&admissionReview)
		if err != nil {
			c.Logger().Errorf("Something went wrong while unmarshalling admission review: %+v", err)
			return c.JSON(http.StatusBadRequest, err)
		}
		c.Logger().Debugf("admission review: %+v", admissionReview)

		pod := v1pod.Pod{}
		if err := json.Unmarshal(admissionReview.Request.Object.Raw, &pod); err != nil {
			c.Logger().Errorf("Something went wrong while unmarshalling pod object: %+v", err)
			return c.JSON(http.StatusBadRequest, err)
		}
		c.Logger().Debugf("pod: %+v", pod)
		c.Logger().Infof("Admission request for pod: %s in namespace: %s", pod.Name, pod.Namespace)

		var admissionReviewResponse v1adm.AdmissionReview
		admissionReviewResponse.TypeMeta.APIVersion = "admission.k8s.io/v1"
		admissionReviewResponse.TypeMeta.Kind = "AdmissionReview"
		admissionReviewResponse.Response = new(v1adm.AdmissionResponse)
		admissionReviewResponse.Response.Allowed = true
		admissionReviewResponse.Response.UID = admissionReview.Request.UID

		if pod.Namespace == "kube-system" {
			c.Logger().Infof("Admitting AKS pod: %+v", pod.Name)
		} else {
			for _, container := range pod.Spec.Containers {
				usingLatest, err := rules.HasLatestTag(container.Image)
				if err != nil {
					c.Logger().Errorf("Error while parsing image name: %+v", err)
					return c.JSON(http.StatusInternalServerError, "error while parsing image name")
				}
				if usingLatest && denyLatestTag {
					admissionReviewResponse.Response.Allowed = false
					admissionReviewResponse.Response.Result = &metav1.Status{
						Message: "Images using latest tag are not allowed",
					}
					c.Logger().Infof("Denied access for image: %+v", container.Image)
					break
				}

				validRegistry, err := rules.IsFromWhiteListedRegistry(
					container.Image,
					registryWhiteList)

				if err != nil {
					c.Logger().Errorf("Error while looking for image registry: %+v", err)
					return c.JSON(
						http.StatusInternalServerError,
						"error while looking for image registry")
				}

				if !validRegistry {
					admissionReviewResponse.Response.Allowed = false
					admissionReviewResponse.Response.Result = &metav1.Status{
						Message: "Images from a non whitelisted registry",
					}
					c.Logger().Infof("Denied access for image: %+v", container.Image)
					break
				}
			}

		}
		c.Logger().Debugf("admission response: %+v", admissionReviewResponse)
		return c.JSON(http.StatusOK, admissionReviewResponse)
	}
}
