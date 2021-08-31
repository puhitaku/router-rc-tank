import json
import time
import uasyncio
from nanoweb.nanoweb import Nanoweb, send_file
from serial.serial import Serial

s = Serial('/dev/ttyACM0', 115200)
app = Nanoweb(port=8080)
op = 's'


def log(fmt, *args):
    t = time.localtime()
    print('[{}]'.format(time.strftime('%Y-%m-%d %H:%M:%S')), fmt.format(*args))


def respond(fn):
    """A mixin decorator to simplify handlers like Flask"""

    async def wrapper(req):
        log('{} {}', req.method, req.url)
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


@app.route('/operation')
@respond
async def operation(req):
    global op

    if req.method == 'GET':
        return 200, {'operation': op, 'error': None}
    elif req.method == 'PUT':
        content_type = req.headers.get('Content-Type')
        if content_type != 'application/json':
            return 400, {'operation': op, 'error': 'bad request, incorrect content type'}

        content_len = req.headers.get('Content-Length')
        if content_len is None:
            return 400, {'operation': op, 'error': 'bad request, has no request body'}

        body_bytes = await req.read(int(content_len))
        body = json.loads(body_bytes.decode())
        if 'operation' not in body:
            return 400, {'operation': op, 'error': 'bad request, lacks operation key'}

        op = body['operation']
        s.write(op)
        return 200, {'operation': op, 'error': None}
    else:
        return 405, {'operation': op, 'error': 'method not allowed'}


@app.route('/healthz')
@respond
async def healthz(req):
    return 200, {'message': "I'm as ready as I'll ever be!"}


async def send_static(req):
    log('{} {}', req.method, req.url)
    await req.write('HTTP/1.1 200 OK\r\n')

    filename = 'static' + req.url

    if req.url in ('/', '/index.html'):
        await req.write('Content-Type: text/html\r\n\r\n')
	filename = 'static/index.html'
    elif req.url == '/simple.html':
        await req.write('Content-Type: text/html\r\n\r\n')
	filename = 'static/simple.html'
    elif req.url.endswith('.css'):
        await req.write('Content-Type: text/css\r\n\r\n')
    elif req.url.endswith('.js'):
        await req.write('Content-Type: application/javascript\r\n\r\n')
    else:
        await req.write('Content-Type: application/octet-stream\r\n\r\n')

    await send_file(req, filename, segment=16384)


app.routes.update(
    {
        '/': send_static,
        '/index.html': send_static,
        '/simple.html': send_static,
        '/assets/*': send_static,
    }
)


loop = uasyncio.get_event_loop()
loop.create_task(app.run())

try:
    loop.run_forever()
finally:
    print('Closing the serial device.')
    s.close()
