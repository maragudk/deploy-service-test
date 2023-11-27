package html

import (
	g "github.com/maragudk/gomponents"
	. "github.com/maragudk/gomponents/html"
)

func HomePage(p PageProps) g.Node {
	p.Title = "Service"
	p.Description = "This is a service."

	return page(p,
		H1(Class("text-2xl font-bold text-gray-900 sm:text-4xl"), g.Text(`This is a Service template.`)),

		P(Class("mt-6 text-lg text-gray-600"), g.Raw(`It's made in Go, and it's really nice.`)),

		P(Class("mt-6 text-lg text-cyan-600 hover:text-cyan-500"), g.Raw(`<a href="https://github.com/maragudk/service">Check out the source code on Github</a>.`)),
	)
}
