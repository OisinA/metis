package api

import (
	"encoding/json"
	"fmt"
	"metis/internal/payload"
	"net/http"

	"github.com/Strum355/log"
)

func (a *API) CreateService(w http.ResponseWriter, r *http.Request) {
	pload := payload.CreateServicePayload{}
	err := json.NewDecoder(r.Body).Decode(&pload)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(400)
		log.WithError(err).Error("Could not decode payload")
		return
	}

	log.WithFields(log.Fields{
		"service": pload.Service.Name(),
	}).Info("Creating service")

	serviceState, err := a.serviceProvider.CreateService(r.Context(), pload.Service)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not create service")
	}

	log.WithFields(log.Fields{
		"service": pload.Service.Name(),
	}).Info("Starting service")

	serviceState, err = a.serviceProvider.StartService(r.Context(), serviceState)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not start service")
	}

	err = json.NewEncoder(w).Encode(payload.CreateServiceResponsePayload{
		ServiceState: serviceState,
	})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not return response")
	}

}

func (a *API) ServiceHealth(w http.ResponseWriter, r *http.Request) {
	pload := payload.ServiceHealthPayload{}
	err := json.NewDecoder(r.Body).Decode(&pload)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(400)
		log.WithError(err).Error("Could not decode payload")
		return
	}

	log.WithFields(log.Fields{
		"service": pload.ServiceState.Service.Name(),
	}).Debug("Updating service health")

	serviceState, err := a.serviceProvider.ServiceHealth(r.Context(), pload.ServiceState)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not get service status")
	}

	err = json.NewEncoder(w).Encode(payload.ServiceHealthResponsePayload{
		ServiceState: serviceState,
	})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not return response")
	}

}

func (a *API) DestroyService(w http.ResponseWriter, r *http.Request) {
	pload := payload.DestroyServicePayload{}
	err := json.NewDecoder(r.Body).Decode(&pload)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(400)
		log.WithError(err).Error("Could not decode payload")
		return
	}

	log.WithFields(log.Fields{
		"service": pload.ServiceState.Service.Name(),
	}).Info("Creating service")

	serviceState, err := a.serviceProvider.StopService(r.Context(), pload.ServiceState)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not stop service")
	}

	log.WithFields(log.Fields{
		"service": pload.ServiceState.Service.Name(),
	}).Info("Starting service")

	serviceState, err = a.serviceProvider.DestroyService(r.Context(), serviceState)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not destroy service")
	}

	err = json.NewEncoder(w).Encode(payload.DestroyServiceResponsePayload{
		ServiceState: serviceState,
	})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		log.WithError(err).Error("Could not return response")
	}

}
