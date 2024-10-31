package server

import (
	"errors"
	"net/http"

	"github.com/jorenkoyen/conter/manifest"
)

func (s *Server) HandleManifestApply(w http.ResponseWriter, r *http.Request) error {
	if !IsJson(r) {
		return errors.New("invalid content type")
	}

	project, err := manifest.Parse(r.Body)
	if err != nil {
		s.logger.Warning("Unable to parse manifest file")
		return err
	}

	s.logger.Debugf("Applying manifest for project=%s", project.Name)
	err = s.Orchestrator.ApplyManifest(r.Context(), project)
	if err != nil {
		s.logger.Warningf("Failed to apply manifest for project=%s: %v", project.Name, err)
		return err
	}

	w.WriteHeader(http.StatusCreated)
	// TODO: response writing
	return nil
}

func (s *Server) HandleManifestDelete(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")
	project := s.Orchestrator.FindManifest(name)
	if project == nil {
		s.logger.Warningf("No manifest found with name=%s", name)
		// TODO: not found error
		return errors.New("manifest not found")
	}

	s.logger.Debugf("Deleting manifest for project=%s", project.Name)
	err := s.Orchestrator.RemoveManifest(r.Context(), project)
	if err != nil {
		s.logger.Warningf("Failed to delete manifest for project=%s: %v", project.Name, err)
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) HandleManifestRetrieve(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}
