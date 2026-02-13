package main

import (
	"context"
	"fmt"
	"net/http"

	"mobile.dani.df/logging"
)

func HttpServer(ctx context.Context) {
	go func() {
		log := ctx.Value("logger").(logging.Logger)

		http.HandleFunc(devicePresentationUrl, func(resp http.ResponseWriter, req *http.Request) { deviceDescriptionHandler(ctx, resp, req) })

		log.Info("[http] Listening for request")
		err := http.ListenAndServe(GetLocalIP()+":8080", nil)
		log.Error("[http] Error occurred while listen and serve: " + err.Error())
	}()
}

func deviceDescriptionHandler(ctx context.Context, response http.ResponseWriter, request *http.Request) {
	log := ctx.Value("logger").(logging.Logger)

	log.Info("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI)

	response.Header().Set("Content-Type", "application/xml")
	response.WriteHeader(http.StatusOK)
	fmt.Fprint(response, rootDevice.StringXML())
}
