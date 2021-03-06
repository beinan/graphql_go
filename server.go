package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/graph-gophers/graphql-go"

	"github.com/beinan/graphql-server/database/mongodb"
	"github.com/beinan/graphql-server/handler"
	"github.com/beinan/graphql-server/loader"
	"github.com/beinan/graphql-server/resolver"
	"github.com/beinan/graphql-server/schema"
	"github.com/beinan/graphql-server/service"
	"github.com/beinan/graphql-server/store"
	"github.com/beinan/graphql-server/utils"
)

var logger = utils.NewLogger()

var db = mongodb.NewDB(logger)

var mongoStore = store.MkMongoStore()
var userService = &service.UserDAO{
	Reader: mongoStore,
	Writer: mongoStore,
}
var authService = &service.AuthDAO{
	Reader: mongoStore,
	Writer: mongoStore,
}
var friendService = &service.FriendRelationDAO{
	Reader: store.RedisStore,
	Writer: store.RedisStore,
}

var services = &service.Services{
	UserService:           userService,
	AuthService:           authService,
	FriendRelationService: friendService,
}

var graphql_schema *graphql.Schema = graphql.MustParseSchema(
	schema.Schema,
	resolver.MkRootResolver(services),
)

func main() {
	logger.Infof("Starting graphql server on %s", ":8888")
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	}))

	latencyStat := handler.LatencyStat(logger)
	dbHandler := handler.DatabaseHandler(db, logger)
	authFilter := handler.AuthFilter(logger)
	loaders := loader.NewLoader(db, logger)
	attachLoader := handler.AttachLoader(loaders)
	graphqlHandler := handler.HandleGraphQL(graphql_schema, logger)
	http.Handle("/query", handler.Chain(latencyStat, dbHandler, authFilter, attachLoader, graphqlHandler))

	logger.Info(http.ListenAndServe(":8888", nil))
}

var page = []byte(`
<!DOCTYPE html>
<html>
	<head>
		<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.10.2/graphiql.css" />
		<script src="https://cdnjs.cloudflare.com/ajax/libs/fetch/1.1.0/fetch.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react-dom.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.10.2/graphiql.js"></script>
	</head>
	<body style="width: 100%; height: 100%; margin: 0; overflow: hidden;">
		<div id="graphiql" style="height: 100vh;">Loading...</div>
		<script>
			function graphQLFetcher(graphQLParams) {
				return fetch("/query", {
					method: "post",
					body: JSON.stringify(graphQLParams),
					credentials: "include",
				}).then(function (response) {
					return response.text();
				}).then(function (responseBody) {
					try {
						return JSON.parse(responseBody);
					} catch (error) {
						return responseBody;
					}
				});
			}

			ReactDOM.render(
				React.createElement(GraphiQL, {fetcher: graphQLFetcher}),
				document.getElementById("graphiql")
			);
		</script>
	</body>
</html>
`)
