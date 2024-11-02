package server

import (
	"encoding/json"
	"errors"
	"github.com/jorenkoyen/conter/manager"
	"github.com/karlseguin/jsonwriter"
	"net/http"
	"time"
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
	name := r.PathValue("name")
	if !s.ContainerManager.DoesProjectExist(name) {
		s.logger.Warningf("No project found with name=%s", name)
		return errors.New("project does not exist")
	}

	status, err := s.ContainerManager.GetProjectStatus(r.Context(), name)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	writer := jsonwriter.New(w)
	writer.RootObject(func() {
		writer.KeyString("project", name)
		writer.Array("services", func() {
			for _, service := range status.Service {
				writer.ArrayObject(func() {
					writer.KeyString("name", service.Name)
					writer.KeyString("hash", service.Hash)
					writer.KeyString("status", status.GetState(service.Name))

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

func (s *Server) HandleProjectList(w http.ResponseWriter, r *http.Request) error {
	projects := s.ContainerManager.FindAllProjects()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	writer := jsonwriter.New(w)
	writer.RootObject(func() {
		writer.KeyInt("total", len(projects))
		writer.Array("projects", func() {
			for name, services := range projects {
				writer.ArrayObject(func() {
					writer.KeyString("name", name)
					writer.KeyValue("running", s.ContainerManager.IsProjectRunning(r.Context(), name))
					writer.Array("services", func() {
						for _, service := range services {
							writer.Value(service.Name)
						}
					})

				})
			}
		})
	})

	return nil
}

func (s *Server) HandleCertificatesRetrieve(w http.ResponseWriter, r *http.Request) error {
	certificates := s.CertificateManager.GetAll()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	writer := jsonwriter.New(w)
	writer.RootObject(func() {
		writer.KeyInt("total", len(certificates))
		writer.Array("certificates", func() {
			for domain, certificate := range certificates {
				writer.ArrayObject(func() {
					writer.KeyString("domain", domain)
					writer.KeyString("challenge", string(certificate.ChallengeType))

					info, err := certificate.Parse()
					if err != nil {
						// skip information
						return
					}

					writer.Object("meta", func() {
						writer.KeyString("subject", info.Subject.CommonName)
						writer.KeyString("issuer", info.Issuer.CommonName)
						writer.KeyString("since", info.NotBefore.Format(time.RFC3339))
						writer.KeyString("expiry", info.NotAfter.Format(time.RFC3339))
						writer.KeyString("serial", info.SerialNumber.String())
						writer.KeyString("signature_algorithm", info.SignatureAlgorithm.String())
						writer.KeyString("public_algorithm", info.PublicKeyAlgorithm.String())
					})
				})
			}
		})
	})

	return nil
}

func (s *Server) HandleCertificateRetrieveData(w http.ResponseWriter, r *http.Request) error {
	domain := r.PathValue("domain")
	cert := s.CertificateManager.Get(domain)
	if cert == nil {
		// TODO: error handling
		return errors.New("not found")
	}

	content, err := cert.CertificateBytes()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
	return nil
}
