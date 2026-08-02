import http from 'k6/http'
import { check, sleep } from 'k6'

export const options = {
  vus: 5,
  duration: '30s',
}

export default function () {
  const res = http.get(`${__ENV.BASE_URL || 'http://127.0.0.1:8080'}/health`)
  check(res, {
    'status is 200': (response) => response.status === 200,
  })
  sleep(1)
}
