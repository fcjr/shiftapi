interface Env {
	ASSETS: Fetcher;
}

export default {
	async fetch(request, env) {
		const url = new URL(request.url);
		if (url.hostname === "shiftapi.dev") {
			url.hostname = "www.shiftapi.dev";
			return Response.redirect(url.toString(), 301);
		}
		const response = await env.ASSETS.fetch(request);
		if (response.status === 404) {
			const notFoundPage = await env.ASSETS.fetch(new URL("/404.html", url.origin));
			return new Response(notFoundPage.body, {
				status: 404,
				headers: notFoundPage.headers,
			});
		}
		return response;
	},
} satisfies ExportedHandler<Env>;
