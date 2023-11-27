package html

import (
	_ "embed"

	g "github.com/maragudk/gomponents"
)

//go:embed terms-of-service.html
var termsOfService string

//go:embed privacy-policy.html
var privacyPolicy string

//go:embed subprocessors.html
var subprocessors string

func TermsOfServicePage(p PageProps) g.Node {
	p.Title = "Terms of Service"
	p.Description = "The Terms of Service for this website and service."
	return page(p,
		prose(
			g.Raw(termsOfService),
			legalDisclaimer(),
		),
	)
}

func PrivacyPolicyPage(p PageProps) g.Node {
	p.Title = "Privacy Policy"
	p.Description = "The Privacy Policy for this website and service."
	return page(p,
		prose(
			g.Raw(privacyPolicy),
			legalDisclaimer(),
		),
	)
}

func SubProcessorsPage(p PageProps) g.Node {
	p.Title = "List of subprocessors"
	p.Description = "The list of subprocessors for this website and service."
	return page(p,
		prose(
			g.Raw(subprocessors),
			legalDisclaimer(),
		),
	)
}

func legalDisclaimer() g.Node {
	return smallProse(
		g.Raw(`This document is licensed under <a href="https://creativecommons.org/licenses/by/4.0/">Creative Commons BY 4.0</a>. Adapted from the <a href="https://github.com/basecamp/policies">Basecamp policies</a> under the same license.`),
	)
}
