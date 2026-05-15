import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

export const status2xx = new Counter('status_2xx');
export const status4xx = new Counter('status_4xx');
export const status5xx = new Counter('status_5xx');
export const statusOther = new Counter('status_other');

export const options = {
  scenarios: {
    fixed_rate: {
      executor: 'constant-arrival-rate',
      rate: 1300,
      timeUnit: '1s',
      duration: '40s',
      preAllocatedVUs: 50,
      maxVUs: 100,
    },
  },
  summaryTrendStats: [
    'avg',
    'min',
    'med',
    'max',
    'p(90)',
    'p(95)',
    'p(99)',
  ],
};

const params = {
  headers: {
    'Content-Type': 'application/json',
  },
};

export default function () {
  const payload = JSON.stringify({
    id: `tx-${__VU}-${__ITER}`,
    transaction: {
      amount: 100.0,
      installments: 1,
      requested_at: '2026-03-11T20:23:35Z',
    },
    customer: {
      avg_amount: 80.0,
      tx_count_24h: 2,
      known_merchants: ['MERC-001'],
    },
    merchant: {
      id: 'MERC-001',
      mcc: '5411',
      avg_amount: 60.0,
    },
    terminal: {
      is_online: false,
      card_present: true,
      km_from_home: 10.0,
    },
    last_transaction: null,
  });

  const res = http.post('http://localhost:9999/fraud-score', payload, params);

  if (res.status >= 200 && res.status < 300) {
    status2xx.add(1);
  } else if (res.status >= 400 && res.status < 500) {
    status4xx.add(1);
  } else if (res.status >= 500 && res.status < 600) {
    status5xx.add(1);
  } else {
    statusOther.add(1);
  }

  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
