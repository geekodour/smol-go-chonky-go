import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
  // A number specifying the number of VUs to run concurrently.
  vus: 10000,
  // A string specifying the total duration of the test run.
  duration: '30s',
};

// export const options = {
//   scenarios: {
//     my_scenario1: {
//       executor: 'constant-arrival-rate',
//       duration: '30s', // total duration
//       preAllocatedVUs: 50, // to allocate runtime resources

//       rate: 50, // number of constant iterations given `timeUnit`
//       timeUnit: '1s',
//     },
//   },
// };

// The function that defines VU logic.
export default function() {
  http.get('http://localhost:8000/');
  // sleep(1);
}
