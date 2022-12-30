package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-nginx/nginx/oam"
	"github.com/layer5io/meshkit/utils"
	"github.com/layer5io/meshkit/utils/kubernetes"
	"github.com/layer5io/meshkit/utils/manifests"
	"github.com/layer5io/meshkit/utils/walker"
	smp "github.com/layer5io/service-mesh-performance/spec"
)

var DefaultVersion string
var DefaultGenerationURL string
var DefaultGenerationMethod string
var WorkloadPath string
var MeshModelPath string

const (
	repo  = "https://helm.nginx.com/stable"
	chart = "nginx-service-mesh"
)

var meshmodelmetadata = map[string]interface{}{
	"Primary Color":   "#009639",
	"Secondary Color": "#42C473",
	"Shape":           "circle",
	"Logo URL":        "",
	"SVG_Color":       "",
	"SVG_White":       "<svg width=\"32\" height=\"32\" viewBox=\"0 0 32 32\" fill=\"none\" xmlns=\"http://www.w3.org/2000/svg\"><path d=\"m21.848 13.38-3.873 1.94c.1.22.141.472.133.723a1.672 1.672 0 0 1-.166.67L21.6 18.75c.331-.38.82-.632 1.36-.67l.166-3.97c-.042-.008-.075-.016-.117-.023a2.022 2.022 0 0 1-1.16-.708Z\" fill=\"#fff\"/><path d=\"M28.425 8.641 16.831 2.502a1.63 1.63 0 0 0-1.543 0L3.702 8.642c-.481.25-.771.714-.771 1.224v12.277a1.26 1.26 0 0 0 .199.708c.132.213.331.395.564.517l11.594 6.139a1.63 1.63 0 0 0 1.543 0l11.594-6.139c.481-.251.771-.715.771-1.225V9.866c0-.502-.29-.974-.77-1.225Zm-4.827 9.357c1.07.16 1.8 1.087 1.626 2.069-.174.98-1.186 1.65-2.256 1.49-1.07-.16-1.8-1.087-1.626-2.069.025-.136.067-.258.117-.38l-3.733-1.917a2.02 2.02 0 0 1-1.584.738 2.039 2.039 0 0 1-1.584-.73l-3.732 1.91a1.65 1.65 0 0 1-.025 1.391l3.799 2.077c.356-.419.92-.685 1.542-.685 1.087 0 1.966.807 1.966 1.803 0 .997-.88 1.803-1.966 1.803-1.086 0-1.965-.806-1.965-1.803 0-.259.058-.502.166-.723l-3.799-2.076a2.015 2.015 0 0 1-1.542.684c-1.087 0-1.966-.806-1.966-1.802 0-.997.879-1.803 1.966-1.803a2.04 2.04 0 0 1 1.584.73l3.732-1.917a1.65 1.65 0 0 1-.141-.67c0-.92.746-1.673 1.716-1.787v-4.237c-.53-.06-.995-.32-1.31-.692l-3.948 2.115c.1.213.158.456.158.7 0 .996-.88 1.802-1.966 1.802-1.086 0-1.965-.806-1.965-1.803 0-.996.879-1.802 1.965-1.802.639 0 1.203.273 1.56.707l3.947-2.115a1.673 1.673 0 0 1-.157-.7c0-.996.879-1.802 1.965-1.802 1.087 0 1.966.806 1.966 1.802 0 .92-.747 1.674-1.717 1.788v4.237c.523.06.987.312 1.294.67l3.798-2.07a1.696 1.696 0 0 1-.124-1.057c.207-.981 1.236-1.62 2.306-1.43 1.07.19 1.766 1.133 1.559 2.115-.166.783-.863 1.354-1.684 1.445v3.978a.166.166 0 0 0 .058.016Z\" fill=\"#fff\"/></svg>",
}

var MeshModelConfig = adapter.MeshModelConfig{ //Move to build/config.go
	Category:    "Orchestration & Management",
	SubCategory: "Service Mesh",
	Metadata:    meshmodelmetadata,
}

// NewConfig creates the configuration for creating components
func NewConfig(version string) manifests.Config {
	return manifests.Config{
		Name:        smp.ServiceMesh_Type_name[int32(smp.ServiceMesh_NGINX_SERVICE_MESH)],
		MeshVersion: version,
		CrdFilter: manifests.NewCueCrdFilter(manifests.ExtractorPaths{
			NamePath:    "spec.names.kind",
			IdPath:      "spec.names.kind",
			VersionPath: "spec.versions[0].name",
			GroupPath:   "spec.group",
			SpecPath:    "spec.versions[0].schema.openAPIV3Schema.properties.spec"}, false),
		ExtractCrds: func(manifest string) []string {
			crds := strings.Split(manifest, "---")
			return crds
		},
	}
}

func getLatestVersion() (string, error) {
	filename := []string{}
	if err := walker.NewGit().
		Owner("nginxinc").
		Repo("helm-charts").
		Branch("master").
		Root("stable/").
		RegisterFileInterceptor(func(f walker.File) error {
			if strings.HasSuffix(f.Name, ".tgz") && strings.HasPrefix(f.Name, "nginx-service-mesh") {
				filename = append(filename, strings.TrimSuffix(strings.TrimPrefix(f.Name, "nginx-service-mesh-"), ".tgz"))
			}
			return nil
		}).Walk(); err != nil {
		return "", err
	}
	filename = utils.SortDottedStringsByDigits(filename)
	if len(filename) == 0 {
		return "", errors.New("no files found")
	}
	return filename[len(filename)-1], nil
}
func init() {
	wd, _ := os.Getwd()
	version, err := getLatestVersion()
	if err != nil {
		fmt.Println("could not get chart version ", err.Error())
		return
	}
	DefaultVersion, err = kubernetes.HelmChartVersionToAppVersion(repo, chart, version)
	if err != nil {
		fmt.Println("could not get version ", err.Error())
		return
	}
	DefaultGenerationURL = "https://github.com/nginxinc/helm-charts/blob/master/stable/nginx-service-mesh-" + version + ".tgz?raw=true"
	DefaultGenerationMethod = adapter.HelmCHARTS
	WorkloadPath = oam.WorkloadPath
	MeshModelPath = filepath.Join(wd, "templates", "meshmodel", "components")
}
