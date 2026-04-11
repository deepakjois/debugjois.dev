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

    if (uri === '/app/transcript-reader') {
        request.uri = '/app/transcript-reader.html';
        return request;
    }

    if (uri === '/app' || uri === '/app/') {
        request.uri = '/app/index.html';
        return request;
    }

    if (!uri.startsWith('/app/')) {
        return request;
    }

    if (uri === '/app/transcript-reader.html' || uri.startsWith('/app/assets/')) {
        return request;
    }

    request.uri = '/app/index.html';

    return request;
}
