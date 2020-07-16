package kates

import (
	"bufio"
	"io"
	"reflect"
	"strings"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

var sch = runtime.NewScheme()

func init() {
	scheme.AddToScheme(sch)
	apiextensions.AddToScheme(sch)
}

func NewObject(kind, version string) (Object, error) {
	return newFromGVK(schema.FromAPIVersionAndKind(version, kind))
}

func newFromGVK(gvk schema.GroupVersionKind) (Object, error) {
	if sch.Recognizes(gvk) {
		robj, err := sch.New(gvk)
		if err != nil {
			return nil, err
		}
		return robj.(Object), nil
	} else {
		un := &Unstructured{}
		un.SetGroupVersionKind(gvk)
		return un, nil
	}
}

func NewUnstructured(kind, version string) *Unstructured {
	uns := &Unstructured{}
	uns.SetGroupVersionKind(schema.FromAPIVersionAndKind(version, kind))
	return uns
}

func ParseManifests(text string) ([]Object, error) {
	yr := utilyaml.NewYAMLReader(bufio.NewReader(strings.NewReader(text)))

	var result []Object

	for {
		bytes, err := yr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.TrimSpace(string(bytes)) == "" {
			continue
		}

		var tm TypeMeta
		err = yaml.Unmarshal(bytes, &tm)
		if err != nil {
			return nil, err
		}

		obj, err := newFromGVK(tm.GroupVersionKind())
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(bytes, obj)
		if err != nil {
			return nil, err
		}

		result = append(result, obj)
	}

	return result, nil
}

func HasOwnerReference(owner, other Object) bool {
	refs := other.GetOwnerReferences()
	for _, r := range refs {
		if r.UID == owner.GetUID() {
			return true
		}
	}
	return false
}

func SetOwnerReferences(owner Object, objects ...Object) {
	gvk := owner.GetObjectKind().GroupVersionKind()
	for _, o := range objects {
		if !HasOwnerReference(owner, o) {
			ref := v1.NewControllerRef(owner, gvk)
			o.SetOwnerReferences(append(o.GetOwnerReferences(), *ref))
		}
	}
}

func ByName(objs interface{}, target interface{}) {
	vobjs := reflect.ValueOf(objs)
	vtarget := reflect.ValueOf(target)
	for i := 0; i < vobjs.Len(); i++ {
		obj := vobjs.Index(i).Interface()
		name := obj.(Object).GetName()
		vtarget.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(obj).Convert(vtarget.Type().Elem()))
	}
}

func MergeUpdate(target *Unstructured, source *Unstructured) {
	annotations := make(map[string]string)
	for k, v := range target.GetAnnotations() {
		annotations[k] = v
	}
	for k, v := range source.GetAnnotations() {
		annotations[k] = v
	}
	target.SetAnnotations(annotations)

	labels := make(map[string]string)
	for k, v := range target.GetLabels() {
		labels[k] = v
	}
	for k, v := range source.GetLabels() {
		labels[k] = v
	}
	target.SetLabels(labels)

	target.SetOwnerReferences(source.GetOwnerReferences())

	spec, ok := source.Object["spec"]
	if ok {
		target.Object["spec"] = spec
	} else {
		delete(target.Object, "spec")
	}
}