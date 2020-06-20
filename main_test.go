package main

import (
	"os"
	"reflect"
	"testing"

	"go.uber.org/zap"
)

func TestMain(t *testing.M) {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	os.Exit(t.Run())
}

func Test_getParamsFromAnnotations(t *testing.T) {
	type args struct {
		nsName      string
		annotations map[string]string
	}
	tests := []struct {
		name   string
		args   args
		want   *parameters
		wantOk bool
	}{
		{
			name: "generic usecase/1",
			args: args{
				nsName: "test",
				annotations: map[string]string{
					"dbinit.k8s.danshin.pro/dbNames":    "test1,test2,test3",
					"dbinit.k8s.danshin.pro/secretName": "testsecret",
					"unrelated/dbNames":                 "test545",
					"unrelated/secretName":              "unrelatedSecretName",
				},
			},
			want: &parameters{
				DBNames:         []string{"test1", "test2", "test3"},
				SecretName:      "testsecret",
				Namespace:       "test",
				SecretNamespace: "default",
			},
			wantOk: true,
		},
		{
			name: "generic usecase/2",
			args: args{
				nsName: "test",
				annotations: map[string]string{
					"dbinit.k8s.danshin.pro/dbNames":    "test1,test2,test3",
					"dbinit.k8s.danshin.pro/secretName": "testsecret",
				},
			},
			want: &parameters{
				DBNames:         []string{"test1", "test2", "test3"},
				SecretName:      "testsecret",
				Namespace:       "test",
				SecretNamespace: "default",
			},
			wantOk: true,
		},
		{
			name: "generic usecase/3",
			args: args{
				nsName:      "test",
				annotations: map[string]string{},
			},
			want: &parameters{
				Namespace:       "test",
				SecretNamespace: "default",
			},
			wantOk: false,
		},
		{
			name: "generic usecase/1",
			args: args{
				nsName: "test",
				annotations: map[string]string{
					"dbinit.k8s.danshin.pro/dbNames":         "test1,test2,test3",
					"dbinit.k8s.danshin.pro/secretName":      "testsecret",
					"dbinit.k8s.danshin.pro/secretNamespace": "somenamespace",
					"unrelated/dbNames":                      "test545",
					"unrelated/secretName":                   "unrelatedSecretName",
				},
			},
			want: &parameters{
				DBNames:         []string{"test1", "test2", "test3"},
				SecretName:      "testsecret",
				Namespace:       "test",
				SecretNamespace: "somenamespace",
			},
			wantOk: true,
		},
		{
			name: "mistakes/1",
			args: args{
				nsName: "test",
				annotations: map[string]string{
					// wrong prefix for one key
					"db-init.k8s.danshin.pro/dbNames":   "test1,test2,test3",
					"dbinit.k8s.danshin.pro/secretName": "testsecret",
				},
			},
			want: &parameters{
				SecretName:      "testsecret",
				Namespace:       "test",
				SecretNamespace: "default",
			},
			wantOk: false,
		},
		{
			name: "mistakes/2",
			args: args{
				nsName: "test",
				annotations: map[string]string{
					// wrong prefix for both keys
					"db-init.k8s.danshin.pro/dbNames":    "test1,test2,test3",
					"db-init.k8s.danshin.pro/secretName": "testsecret",
				},
			},
			want: &parameters{
				Namespace:       "test",
				SecretNamespace: "default",
			},
			wantOk: false,
		},
		{
			name: "mistakes/3",
			args: args{
				nsName: "test",
				annotations: map[string]string{
					// wrong key name for both keys
					"dbinit.k8s.danshin.pro/dbNames1":    "test1,test2,test3",
					"dbinit.k8s.danshin.pro/secretName1": "testsecret",
				},
			},
			want: &parameters{
				Namespace:       "test",
				SecretNamespace: "default",
			},
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := getParamsFromAnnotations(tt.args.nsName, tt.args.annotations); !reflect.DeepEqual(got, tt.want) || ok != tt.wantOk {
				t.Errorf("getParamsFromAnnotations() = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOk)
			}
		})
	}
}
