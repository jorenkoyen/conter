package server

import (
	"encoding/json"
	"errors"
	"github.com/jorenkoyen/conter/manager"
	"github.com/karlseguin/jsonwriter"
	"net/http"
)

func (s *Server) HandleProjectApply(w http.ResponseWriter, r *http.Request) error {
	if !IsJson(r) {
		return errors.New("invalid content type")
	}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	opts := new(manager.ApplyProjectOptions)
	if err := decoder.Decode(opts); err != nil {
		return err
	}

	applied, err := s.ContainerManager.ApplyProject(r.Context(), opts)
	if err != nil {
		s.logger.Warningf("Failed to apply configuration for project=%s: %v", opts.ProjectName, err)
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	writer := jsonwriter.New(w)
	writer.RootObject(func() {
		writer.KeyString("project", opts.ProjectName)
		writer.Array("services", func() {
			for _, service := range applied {
				writer.ArrayObject(func() {
					writer.KeyString("name", service.Name)
					writer.KeyString("hash", service.Hash)

					if service.Ingress.Domain != "" {
						writer.Object("ingress", func() {
							writer.KeyString("domain", service.Ingress.Domain)
							writer.KeyString("internal", service.Ingress.TargetEndpoint)
							writer.KeyString("challenge", string(service.Ingress.ChallengeType))
						})
					}
				})
			}
		})
	})
	return nil
}

func (s *Server) HandleProjectDelete(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")
	if !s.ContainerManager.DoesProjectExist(name) {
		s.logger.Warningf("No project found with name=%s", name)
		return errors.New("project does not exist")
	}

	err := s.ContainerManager.RemoveProject(r.Context(), name)
	if err != nil {
		s.logger.Warningf("Failed to delete configuration for project=%s: %v", name, err)
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) HandleProjectRetrieve(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}
