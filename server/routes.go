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
					writer.KeyString("status", manager.StatusRunning) // always running when applied

					if service.IsExposed() {
						writer.Object("ingress", func() {
							writer.KeyString("internal", service.Ingress.TargetEndpoint)
							writer.KeyString("challenge", string(service.Ingress.ChallengeType))
							writer.Array("domains", func() {
								for _, domain := range service.Ingress.Domains {
									writer.Value(domain)
								}
							})
						})
					}

					if len(service.Volumes) > 0 {
						writer.Array("volumes", func() {
							for _, volume := range service.Volumes {
								writer.Value(volume.Path)
							}
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
			for _, service := range status.Services {
				writer.ArrayObject(func() {
					writer.KeyString("name", service.Name)
					writer.KeyString("hash", service.Hash)
					writer.KeyString("status", status.GetState(service.Name))

					if service.IsExposed() {
						writer.Object("ingress", func() {
							writer.KeyString("internal", service.Ingress.TargetEndpoint)
							writer.KeyString("challenge", string(service.Ingress.ChallengeType))
							writer.Array("domains", func() {
								for _, domain := range service.Ingress.Domains {
									writer.Value(domain)
								}
							})
						})
					}

					if len(service.Volumes) > 0 {
						writer.Array("volumes", func() {
							for _, volume := range service.Volumes {
								writer.Value(volume.Path)
							}
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
	writer.RootArray(func() {
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

	return nil
}

func (s *Server) HandleCertificatesRetrieve(w http.ResponseWriter, r *http.Request) error {
	certificates := s.CertificateManager.GetAll()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	writer := jsonwriter.New(w)
	writer.RootArray(func() {
		for _, certificate := range certificates {
			writer.ArrayObject(func() {
				writer.KeyString("id", certificate.ID)
				writer.KeyString("challenge", string(certificate.ChallengeType))
				writer.Array("domains", func() {
					for _, domain := range certificate.Domains {
						writer.Value(domain)
					}
				})

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

	return nil
}

func (s *Server) HandleCertificateRetrieveData(w http.ResponseWriter, r *http.Request) error {
	domain := r.PathValue("domain")
	cert := s.CertificateManager.Get(domain)
	if cert == nil {
		return errors.New("not found")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	writer := jsonwriter.New(w)
	writer.RootObject(func() {
		writer.KeyString("id", cert.ID)
		writer.KeyString("challenge", string(cert.ChallengeType))
		writer.KeyString("pem", cert.Certificate)
		writer.Array("domains", func() {
			for _, d := range cert.Domains {
				writer.Value(d)
			}
		})

		if info, err := cert.Parse(); err == nil {
			writer.Object("meta", func() {
				writer.KeyString("subject", info.Subject.CommonName)
				writer.KeyString("issuer", info.Issuer.CommonName)
				writer.KeyString("since", info.NotBefore.Format(time.RFC3339))
				writer.KeyString("expiry", info.NotAfter.Format(time.RFC3339))
				writer.KeyString("serial", info.SerialNumber.String())
				writer.KeyString("signature_algorithm", info.SignatureAlgorithm.String())
				writer.KeyString("public_algorithm", info.PublicKeyAlgorithm.String())
			})
		}
	})
	return nil
}

func (s *Server) HandleCertificateRenew(w http.ResponseWriter, r *http.Request) error {
	domain := r.PathValue("domain")
	cert := s.CertificateManager.Get(domain)
	if cert == nil {
		s.logger.Warningf("No certificate found for domain=%s when trying to renew", domain)
		return errors.New("not found")
	}

	err := s.CertificateManager.ChallengeCreate([]string{domain}, cert.ChallengeType)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusAccepted)
	return nil
}

func (s *Server) HandleSystemTask(w http.ResponseWriter, r *http.Request) error {
	task := r.PathValue("task")
	if task == "" {
		return errors.New("missing task parameter value")
	}

	if task == "batch_certificates" {
		s.CertificateManager.Batch()
	} else {
		return errors.New("unable to handle task, unknown to system")
	}

	w.WriteHeader(http.StatusOK)
	return nil
}
