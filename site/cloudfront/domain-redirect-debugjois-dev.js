function handler(event) {
    var request = event.request;
    var uri = request.uri;

    if (request.headers.host.value === 'debugjois.dev') {
        return {
            statusCode: 301,
            statusDescription: 'Moved Permanently',
            headers: {
                location: { value: 'https://www.debugjois.dev' + uri }
            }
        };
    }

    return request;
}
