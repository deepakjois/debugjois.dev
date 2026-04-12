function handler(event) {
    var request = event.request;
    var uri = request.uri;

    function hasFileExtension(path) {
        return /\.[^/]+$/.test(path);
    }

    function redirect(path) {
        return {
            statusCode: 301,
            statusDescription: 'Moved Permanently',
            headers: {
                location: { value: path }
            }
        };
    }

    if (request.headers.host.value === 'debugjois.dev') {
        return {
            statusCode: 301,
            statusDescription: 'Moved Permanently',
            headers: {
                location: { value: 'https://www.debugjois.dev' + uri }
            }
        };
    }

    if (uri === '/apps/spa/index.html') {
        return redirect('/apps/spa');
    }

    if (uri === '/apps/transcript-reader/index.html') {
        return redirect('/apps/transcript-reader');
    }

    if (uri === '/apps/spa/' || uri === '/apps/transcript-reader/') {
        return redirect(uri.slice(0, -1));
    }

    if (uri === '/apps/transcript-reader') {
        request.uri = '/apps/transcript-reader/index.html';
        return request;
    }

    if (uri === '/apps/spa') {
        request.uri = '/apps/spa/index.html';
        return request;
    }

    if (!uri.startsWith('/apps/')) {
        return request;
    }

    if (uri.startsWith('/apps/assets/') || hasFileExtension(uri)) {
        return request;
    }

    if (uri.startsWith('/apps/spa/')) {
        request.uri = '/apps/spa/index.html';
    }

    return request;
}
