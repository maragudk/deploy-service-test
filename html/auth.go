package html

import (
	g "github.com/maragudk/gomponents"
	. "github.com/maragudk/gomponents/html"

	"github.com/maragudk/service/model"
)

func SignupPage(p PageProps, newUser model.User) g.Node {
	p.Title = "Sign up"

	return page(p,
		authPageCard(
			FormEl(Action("/signup"), Method("post"), Class("space-y-6"),
				Div(Class("text-center"),
					h1(g.Text(`Sign up`)),
					a(Href("/login"), g.Text("or log in instead")),
				),

				g.If(newUser.Email.String() != "",
					alertBox(g.Raw(`There's already a user with that email address. `), a(Href("/login"), g.Text("Log in instead?"))),
				),

				Div(
					label("name", "Name"),
					input(Type("text"), ID("name"), Name("name"), Value(newUser.Name), AutoComplete("name"),
						Placeholder("Me"), Required(), g.If(newUser.Name == "", AutoFocus())),
				),

				Div(
					label("email", "Email"),
					input(Type("email"), ID("email"), Name("email"), Value(newUser.Email.String()), AutoComplete("email"),
						Placeholder("me@example.com"), Required(), g.If(newUser.Name != "", AutoFocus())),
				),

				Div(Class("flex items-center space-x-2"),
					Input(ID("accept"), Name("accept"), Type("checkbox"), Value("true"), Required(),
						Class("h-4 w-4 rounded border-gray-300 text-cyan-600 focus:ring-cyan-500")),
					Label(For("accept"), Class("text-gray-900"),
						g.Text(`I accept the `),
						a(Href("/legal/terms-of-service"), Target("_blank"), g.Text(`Terms of Service`)),
						g.Text(` and `),
						a(Href("/legal/privacy-policy"), Target("_blank"), g.Text(`Privacy Policy`)),
						g.Text(`.`),
					),
				),

				button(Type("submit"), g.Text(`Sign up`)),
			),
		),
	)
}

func SignupThanksPage(props PageProps) g.Node {
	props.Title = "Thank you for signing up"

	return page(props,
		authPageCard(
			Div(Class("text-center"),
				h1(g.Text(`Check your email`)),
			),

			p("mt-8", g.Raw(`Thank you for signing up! 😊`)),
			p("mt-4", g.Raw(`Now check your email and click the link in it.`)),
		),
	)
}

func LoginPage(p PageProps, email string) g.Node {
	p.Title = "Log in"

	return page(p,
		authPageCard(
			FormEl(Action("/login/email"), Method("post"), Class("space-y-6"),
				Div(Class("text-center"),
					h1(g.Text(`Log in`)),
					a(Href("/signup"), g.Text("or sign up instead")),
				),

				g.If(email != "",
					alertBox(g.Raw(`It doesn't look like anyone's signed up with that email address. `), a(Href("/signup"), g.Text("Sign up instead?"))),
				),

				Div(
					label("email", "Email"),
					input(Type("email"), ID("email"), Name("email"), AutoComplete("email"), Placeholder("me@example.com"), Required(), AutoFocus()),
				),

				button(Type("submit"), g.Text(`Log in`)),
			),
		),
	)
}

func LoginTokenPage(props PageProps, token string) g.Node {
	props.Title = "Log in"

	return page(props,
		authPageCard(
			FormEl(Action("/login/token"), Method("post"),
				Div(Class("text-center"),
					h1(g.Text(`Log in`)),
				),

				p("mt-8 mb-4", g.Raw(`Click the button to log in. 😊`)),

				Input(Type("hidden"), Name("token"), Value(token)),

				button(Type("submit"), g.Text(`Log in`)),
			),
		),
	)
}

func LoginCheckPage(props PageProps) g.Node {
	props.Title = "Check your email"

	return page(props,
		authPageCard(
			Div(Class("text-center"),
				h1(g.Text(`Check your email`)),
			),

			p("mt-8", g.Raw(`Now check your email and click the link in it.`)),
		),
	)
}

func UserInactivePage(props PageProps) g.Node {
	props.Title = "Your user is inactive"

	return page(props,
		authPageCard(
			Div(Class("text-center"),
				h1(g.Text(`Account inactive`)),
			),

			p("mt-8", g.Raw(`Your account is inactive. If you think this is an error, `),
				a(Href("mailto:support@maragu.dk"), g.Text("reach out to support")), g.Text(".")),
		),
	)
}

func TokenExpiredPage(props PageProps) g.Node {
	props.Title = "Your link has expired"

	return page(props,
		authPageCard(
			Div(Class("text-center"),
				h1(g.Text(`Link expired`)),
			),

			p("mt-8", g.Raw(`Your link has expired. `), a(Href("/login"), g.Text("Log in again")), g.Text(".")),
		),
	)
}

func authPageCard(children ...g.Node) g.Node {
	return Div(Class("sm:mx-auto sm:w-full sm:max-w-md"),
		card(g.Group(children)),
	)
}
