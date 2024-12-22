package manager

import (
	"fmt"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"slices"
)

type IngressManager struct {
	logger             *logger.Logger
	Database           *db.Client
	CertificateManager *CertificateManager
}

// NewIngressManager creates a new instance for managing routes for directing traffic to the correct container.
func NewIngressManager() *IngressManager {
	return &IngressManager{
		logger: log.WithName("ingress-mgr"),
	}
}

// RegisterRoute will register a new ingress route and complete the necessary actions to make it ready for use.
func (i *IngressManager) RegisterRoute(ingress types.Ingress) error {
	i.logger.Debugf("Registering route for %s (endpoint=%s, challenge=%s)", ingress.String(), ingress.TargetEndpoint, ingress.ChallengeType)

	// check all domains linked to ingress
	//	-> fail if any exist that are not for the same project / service.
	for _, domain := range ingress.Domains {
		match, err := i.Match(domain)
		if err == nil {
			if match.TargetProject != ingress.TargetProject {
				return fmt.Errorf("domain=%s is already used for project=%s", domain, match.TargetProject)
			}
			if match.TargetService != ingress.TargetService {
				return fmt.Errorf("domain=%s is already used for service=%s", domain, match.TargetService)
			}
		}
	}

	// save ingress routes
	i.logger.Infof("Registering %s", ingress.String())
	err := i.Database.SaveIngressRoute(&ingress) // creates an entry for each domain specified
	if err != nil {
		return fmt.Errorf("failed to save ingress route: %w", err)
	}

	// start certificate creation
	err = i.CertificateManager.ChallengeCreate(ingress.Domains, ingress.ChallengeType)
	if err != nil {
		return fmt.Errorf("failed to create certificates for %s", ingress.String())
	}

	return nil
}

// RemoveUnusedRoutes will remove all unused routes related to the specified project.
func (i *IngressManager) RemoveUnusedRoutes(project string, excludedDomains []string) (int, error) {
	i.logger.Tracef("Removing unused routes for project=%s (excluded=%s)", project, excludedDomains)

	routes := i.Database.GetIngressRoutesByProject(project)
	removed := 0
	for domain, _ := range routes {
		if len(excludedDomains) == 0 || slices.Index(excludedDomains, domain) == -1 {
			// not listed in excluded domains -> remove
			i.logger.Debugf("Removing unused route for domain=%s linked to project=%s", domain, project)
			err := i.Database.RemoveIngressRoute(domain)
			if err != nil {
				return removed, fmt.Errorf("failed to remove %s: %w", domain, err)
			}

			removed++
		}
	}

	return removed, nil
}

// RemoveAllRoutes will remove all routes linked to the specified project.
func (i *IngressManager) RemoveAllRoutes(project string) (int, error) {
	return i.RemoveUnusedRoutes(project, nil) // no excluded domains
}

// Match will retrieve the ingress route information for the specified domain.
func (i *IngressManager) Match(domain string) (*types.Ingress, error) {
	return i.Database.GetIngressRoute(domain)
}
