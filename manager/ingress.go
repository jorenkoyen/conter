package manager

import (
	"context"
	"fmt"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manifest"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"slices"
)

type Ingress struct {
	logger   *logger.Logger
	Database *db.Client
}

func NewIngressManager() *Ingress {
	return &Ingress{
		logger: log.WithName("ingress"),
	}
}

type RegisterRouteOptions struct {
	Challenge manifest.ChallengeType
	Project   string
	Service   string
}

// RegisterRoute will register a new ingress route and complete the necessary actions to make it ready for use.
func (i *Ingress) RegisterRoute(ctx context.Context, domain string, endpoint string, opts RegisterRouteOptions) error {
	i.logger.Debugf("Registering route for domain=%s (endpoint=%s, challenge_type=%s)", domain, endpoint, opts.Challenge)

	// TODO: check if we already have a registered route
	//	-> update endpoint (if required)
	// 	-> check if project & service are correct

	// TODO: create challenge (http01 only for now)

	// register route
	route := &manifest.IngressRoute{Domain: domain, Endpoint: endpoint}
	return i.Database.SaveIngressRoute(route)
}

// RemoveUnusedRoutes will remove all unused routes related to the specified project.
func (i *Ingress) RemoveUnusedRoutes(project string, excludedDomains []string) error {
	i.logger.Tracef("Removing unused routes for project=%s (excluded=%s)", project, excludedDomains)

	routes := i.Database.GetIngressRoutesByProject(project)
	for _, route := range routes {
		if slices.Index(excludedDomains, route.Domain) == -1 {
			// not listed in excluded domains -> remove
			i.logger.Debugf("Removing unused route for domain=%s linked to project=%s", route.Domain, project)
			err := i.Database.RemoveIngressRoute(route.Domain)
			if err != nil {
				return fmt.Errorf("failed to remove %s: %w", route.Domain, err)
			}
		}
	}

	return nil
}

// RemoveAllRoutes will remove all routes linked to the specified project.
func (i *Ingress) RemoveAllRoutes(project string) error {
	return i.RemoveUnusedRoutes(project, []string{}) // no excluded domains
}

// Match will retrieve the ingress route information for the specified domain.
func (i *Ingress) Match(domain string) (*manifest.IngressRoute, error) {
	return i.Database.GetIngressRoute(domain)
}
