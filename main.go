package main

import (
	"context"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gobuffalo/pop"
)

type parameters struct {
	Namespace       string
	DBNames         []string `annotation:"dbinit.k8s.danshin.pro/dbNames"`
	SecretName      string   `annotation:"dbinit.k8s.danshin.pro/secretName"`
	SecretNamespace string   `annotation:"dbinit.k8s.danshin.pro/secretNamespace"`
}

func getConfig() (*rest.Config, error) {
	if kubecfg := os.Getenv("KUBECONFIG"); len(kubecfg) > 0 {
		zap.L().Info("using KUBECONFIG", zap.String("path", kubecfg))
		return clientcmd.BuildConfigFromFlags("", path.Clean(kubecfg))
	}

	zap.L().Info("using in-cluster config")
	return rest.InClusterConfig()
}

func getClient() (*kubernetes.Clientset, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func getStringFromAnnotations(annotations map[string]string, key string) string {
	if annotations == nil {
		return ""
	}
	raw, ok := annotations[key]
	if !ok {
		return ""
	}
	return raw
}

func getListFromAnnotations(annotations map[string]string, key string) []string {
	raw := getStringFromAnnotations(annotations, key)
	if raw == "" {
		return nil
	}

	list := strings.Split(raw, ",")
	for i := range list {
		list[i] = strings.TrimSpace(list[i])
	}

	return list
}

func checkRequirements(params *parameters) (ok bool) {
	gotDBNames := len(params.DBNames) > 0
	gotSecretName := len(params.SecretName) > 0
	ok = gotDBNames && gotSecretName

	if !ok {
		if !gotDBNames && gotSecretName {
			zap.L().Warn(
				"ignoring db-init annotations due to missing dbNames: check the spelling, prefix, and separator, "+
					"which must be a single comma without spaces",

				zap.String("namespace", params.Namespace),
			)
		}
		if !gotSecretName && gotDBNames {
			zap.L().Warn(
				"ignoring db-init annotations due to missing secretName: check the spelling, prefix, and separator, "+
					"which must be a single comma without spaces",

				zap.String("namespace", params.Namespace),
			)
		}
	}
	return
}

func getParamsFromAnnotations(nsName string, annotations map[string]string) (*parameters, bool) {
	paramType := reflect.TypeOf(parameters{})
	numFields := paramType.NumField()
	res := reflect.New(paramType)

	for i := 0; i < numFields; i++ {
		field := paramType.Field(i)
		keyTag := field.Tag.Get("annotation")
		if keyTag == "" {
			continue
		}

		switch field.Type.Kind() {
		case reflect.Slice:
			list := getListFromAnnotations(annotations, keyTag)
			res.Elem().Field(i).Set(reflect.ValueOf(list))
		case reflect.String:
			str := getStringFromAnnotations(annotations, keyTag)
			res.Elem().Field(i).Set(reflect.ValueOf(str))
		}
	}

	params := res.Interface().(*parameters)
	params.Namespace = nsName
	ok := checkRequirements(params)
	if ok && params.SecretNamespace == "" {
		zap.L().Info(
			"using 'default' namespace as no secretNamespace were provided",
			zap.String("namespace", params.Namespace),
		)
		params.SecretNamespace = "default"
	}

	return params, ok
}

func initDatabases(clientset *kubernetes.Clientset, params *parameters) {
	secret, err := clientset.CoreV1().Secrets(params.Namespace).Get(context.Background(), params.SecretName, metav1.GetOptions{})
	if err != nil {
		zap.L().Error(
			"could not get secret to initialize databases",
			zap.String("namespace", params.Namespace),
			zap.String("secretName", params.SecretName),
			zap.Strings("dbNames", params.DBNames),
		)
		return
	}

	if secret.StringData == nil || secret.StringData["dsn"] == "" {
		zap.L().Error(
			"could not init database: secret must have stringData with a non-empty 'dsn' field in format supported by github.com/gobuffalo/pop",
			zap.String("namespace", params.Namespace),
		)
		return
	}

	dsn := secret.StringData["dsn"]
	if strings.HasPrefix(dsn, "sqlite") {
		zap.L().Error(
			"sqlite initialization is not supported",
			zap.String("namespace", params.Namespace),
			zap.String("secretName", params.SecretName),
			zap.Strings("dbNames", params.DBNames),
		)
		return
	}

	deets := &pop.ConnectionDetails{
		URL: dsn,
	}
	err = deets.Finalize()
	if err != nil {
		zap.L().Error(
			"invalid dsn",
			zap.String("namespace", params.Namespace),
			zap.String("secretName", params.SecretName),
			zap.Strings("dbNames", params.DBNames),
			zap.Error(err),
		)
		return
	}
	for _, dbName := range params.DBNames {
		deets.Database = dbName
		conn, err := pop.NewConnection(deets)

		if err != nil {
			zap.L().Error(
				"could not connect to the database",
				zap.String("namespace", params.Namespace),
				zap.String("secretName", params.SecretName),
				zap.Strings("dbNames", params.DBNames),
				zap.Error(err),
			)
			return
		}
		defer conn.Close()

		err = pop.CreateDB(conn)
		if err != nil {
			zap.L().Error(
				"could not create database",
				zap.String("namespace", params.Namespace),
				zap.String("secretName", params.SecretName),
				zap.Strings("dbNames", params.DBNames),
				zap.Error(err),
			)
		}
	}
}

func watchNamespaceAnnotations(clientset *kubernetes.Clientset) {
	for range time.NewTicker(15 * time.Second).C {
		namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			zap.L().Error("could not fetch namespaces", zap.Error(err))
		}
		for _, ns := range namespaces.Items {
			params, ok := getParamsFromAnnotations(ns.Name, ns.Annotations)
			if !ok {
				continue
			}

			initDatabases(clientset, params)
		}
	}
}

func main() {
	{
		l, _ := zap.NewProduction()
		zap.ReplaceGlobals(l)
	}

	clientset, err := getClient()
	if err != nil {
		zap.L().Fatal("cannot connect to k8s api", zap.Error(err))
	}

	go watchNamespaceAnnotations(clientset)

	<-make(chan struct{})
}
