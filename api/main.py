import json
import uasyncio
from nanoweb.nanoweb import Nanoweb

app = Nanoweb(port=8080)


def wrap(fn):
    """A mixin decorator to simplify handlers like Flask"""

    async def wrapper(req):
        res = await fn(req)

        if isinstance(res, tuple):
            # Tuple = a tuple of status code and the body
            status, body = res
        else:
            # Others = implies "200 OK"
            status, body = 200, res

        # Start writing the response header
        await req.write('HTTP/1.1 {}\r\n'.format(status))

        if isinstance(body, dict) or isinstance(body, list):
            # Dict or list = jsonified
            await req.write('Content-Type: application/json\r\n\r\n')
            await req.write(json.dumps(body))
        else:
            # Others = implies a plain text and be transmitted as-is
            await req.write('Content-Type: text/plain\r\n\r\n')
            await req.write(body)

    return wrapper


@app.route('/ping')
@wrap
async def ping(req):
    return 200, 'pong'


@app.route('/healthz')
@wrap
async def ping(req):
    return 200, {'message': "I'm as ready as I'll ever be!"}


# Static files
app.routes.update(
    {
        '/index.html': ('./static/index.html', {'name': 'Happy Router'}),
    }
)


loop = uasyncio.get_event_loop()
loop.create_task(app.run())
loop.run_forever()
