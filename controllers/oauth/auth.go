package oauth

import (
	"net/http"

	"github.com/NyaaPantsu/nyaa/controllers/router"
	"github.com/NyaaPantsu/nyaa/templates"
	"github.com/NyaaPantsu/nyaa/utils/cookies"
	"github.com/NyaaPantsu/nyaa/utils/log"
	msg "github.com/NyaaPantsu/nyaa/utils/messages"
	"github.com/NyaaPantsu/nyaa/utils/oauth2"
	"github.com/NyaaPantsu/nyaa/utils/validator"
	"github.com/NyaaPantsu/nyaa/utils/validator/user"
	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
)

func authEndpoint(c *gin.Context) {
	// This context will be passed to all methods.
	ctx := fosite.NewContext()
	// Let's create an AuthorizeRequest object!
	// It will analyze the request and extract important information like scopes, response type and others.
	ar, err := oauth2.Oauth2.NewAuthorizeRequest(ctx, c.Request)
	if err != nil {
		log.Errorf("Error occurred in NewAuthorizeRequest: %s", err)
		oauth2.Oauth2.WriteAuthorizeError(c.Writer, ar, err)
		return
	}
	// We add needed template variables
	templateVariables := templates.Commonvariables(c)
	templateVariables.Set("Scopes", ar.GetRequestedScopes())
	templateVariables.Set("Client", ar.GetClient())

	// Get the current user
	user := router.GetUser(c)
	// If not already logged in
	if user.ID == 0 {
		// We display the login form and log him in
		b := userValidator.LoginForm{}
		messages := msg.GetMessages(c)
		c.Bind(&b)
		validator.ValidateForm(&b, messages)
		if messages.HasErrors() {
			templates.Render(c, "site/api/login.jet.html", templateVariables)
			return
		}

		_, _, errorUser := cookies.CreateUserAuthentication(c, &b)
		if errorUser != nil {
			templates.Render(c, "site/api/login.jet.html", templateVariables)
			return
		}
	}

	// if logged in but grant form wasn't sent, we show grant form and stop the request
	if c.PostForm("grant") == "" {
		templates.Render(c, "site/api/grant.jet.html", templateVariables)
		return
	}

	// if logged in and grant form sent
	// let's see what scopes the user gave consent to
	for _, scope := range c.PostFormArray("scopes") {
		ar.GrantScope(scope)
	}
	client, err := oauth2.Store.GetConcreteClient(ar.GetClient().GetID())
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	mySessionData := oauth2.NewSession(user.Username, client.ClientURI)
	response, err := oauth2.Oauth2.NewAuthorizeResponse(ctx, ar, mySessionData)
	// Catch any errors, e.g.:
	// * unknown client
	// * invalid redirect
	// * ...
	if err != nil {
		log.Errorf("Error occurred in NewAuthorizeResponse: %s\n", err)
		oauth2.Oauth2.WriteAuthorizeError(c.Writer, ar, err)
		return
	}

	// Last but not least, send the response!
	oauth2.Oauth2.WriteAuthorizeResponse(c.Writer, ar, response)
}
