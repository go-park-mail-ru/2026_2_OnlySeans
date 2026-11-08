package assets

import (
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gorilla/mux"
)

func Register(client *s3.Client, bucket string) func(*mux.Router) {
	return func(r *mux.Router) {
		r.HandleFunc("/static/{key:.*}", func(w http.ResponseWriter, r *http.Request) {
			key := mux.Vars(r)["key"]
			if key == "" || strings.Contains(key, "..") {
				http.NotFound(w, r)
				return
			}
			object, err := client.GetObject(r.Context(), &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer object.Body.Close()
			if object.ContentType != nil {
				w.Header().Set("Content-Type", *object.ContentType)
			}
			w.Header().Set("Cache-Control", "public, max-age=3600")
			io.Copy(w, object.Body)
		}).Methods(http.MethodGet)
	}
}
