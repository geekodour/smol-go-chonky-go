import http from 'k6/http';

export const options = {
  vus: 1000,
  duration: '10s',
  thresholds: {
    http_req_duration: ['p(95)<200'], //95% of requests should be below 200ms
  },
};

export default function() {
  http.get('http://localhost:8000/cats');
}
