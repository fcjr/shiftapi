export default {
	async fetch(request) {
		const url = new URL(request.url);
		if (url.hostname === "shiftapi.dev") {
			url.hostname = "www.shiftapi.dev";
			return Response.redirect(url.toString(), 301);
		}
		return new Response("Not found", { status: 404 });
	},
} satisfies ExportedHandler;
