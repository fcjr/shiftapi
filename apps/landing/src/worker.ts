interface Env {
	ASSETS: Fetcher;
}

export default {
	async fetch(request: Request, env: Env) {
		const url = new URL(request.url);
		if (url.hostname === "shiftapi.dev") {
			url.hostname = "www.shiftapi.dev";
			return Response.redirect(url.toString(), 301);
		}
		return env.ASSETS.fetch(request);
	},
} satisfies ExportedHandler<Env>;
