package server

import (
	"errors"
	"net/http"

	"github.com/jorenkoyen/conter/model"
)

func (s *Server) HandleProjectApply(w http.ResponseWriter, r *http.Request) error {
	if !IsJson(r) {
		return errors.New("invalid content type")
	}

	project, err := model.ParseProject(r.Body)
	if err != nil {
		s.logger.Warning("Unable to parse project file")
		return err
	}

	err = s.Orchestrator.ApplyProject(r.Context(), project)
	if err != nil {
		s.logger.Warningf("Failed to apply configuration for project=%s: %v", project.Name, err)
		return err
	}

	w.WriteHeader(http.StatusCreated)
	// TODO: response writing
	return nil
}

func (s *Server) HandleProjectDelete(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")
	project := s.Orchestrator.FindProject(name)
	if project == nil {
		s.logger.Warningf("No project found with name=%s", name)
		// TODO: not found error
		return errors.New("project not found")
	}

	err := s.Orchestrator.RemoveProject(r.Context(), project)
	if err != nil {
		s.logger.Warningf("Failed to delete configuration for project=%s: %v", project.Name, err)
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) HandleProjectRetrieve(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}
