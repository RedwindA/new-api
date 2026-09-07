/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { Banner, Button, Modal, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers';
import { auditResultLabel } from './result';

function BodyBlock(props) {
  const { t } = useTranslation();
  let body = props.value || '-';
  try {
    body = JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    // The server also returns omission notices and empty bodies.
  }
  return (
    <section className='space-y-2'>
      <div className='flex items-center gap-2'>
        <h3 className='font-medium'>{props.label}</h3>
        {props.truncated && (
          <span className='text-[var(--semi-color-warning)]'>
            {t('Truncated')}
          </span>
        )}
      </div>
      <pre className='max-h-64 overflow-auto whitespace-pre-wrap break-all rounded bg-[var(--semi-color-fill-0)] p-3 text-xs'>
        {body}
      </pre>
    </section>
  );
}

export default function AuditRequestDetail(props) {
  const { t } = useTranslation();
  const requestId = props.log?.request_id;
  const createdAt = props.log?.created_at;
  const [state, setState] = useState({ loading: true });
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    if (!requestId) return;
    const controller = new AbortController();
    setState({ loading: true });
    API.get('/api/audit_request/detail', {
      params: { request_id: requestId, created_at: createdAt },
      signal: controller.signal,
      disableDuplicate: true,
    })
      .then(({ data }) => {
        if (controller.signal.aborted) return;
        if (!data.success || !data.data) {
          setState({
            error: data.message || t('Failed to load request details'),
          });
          return;
        }
        setState({ detail: data.data });
      })
      .catch(() => {
        if (!controller.signal.aborted)
          setState({ error: t('Failed to load request details') });
      });
    return () => controller.abort();
  }, [requestId, createdAt, retry, t]);

  const detail = state.detail;
  return (
    <Modal
      visible={Boolean(props.log)}
      title={t('Request details')}
      onCancel={props.onClose}
      footer={null}
      style={{ width: 'min(900px, calc(100vw - 32px))' }}
    >
      <div className='max-h-[75vh] space-y-4 overflow-auto py-2'>
        {state.loading && <Spin tip={t('Loading details...')} />}
        {state.error && (
          <Banner
            type='danger'
            closeIcon={null}
            description={state.error}
            fullMode={false}
          />
        )}
        {state.error && (
          <Button onClick={() => setRetry((value) => value + 1)}>
            {t('Retry')}
          </Button>
        )}
        {detail && (
          <>
            <dl className='grid grid-cols-1 gap-3 text-sm sm:grid-cols-2'>
              {[
                [t('Request ID'), detail.request_id],
                [
                  t('Time'),
                  new Date(detail.created_at * 1000).toLocaleString(),
                ],
                [t('IP'), detail.ip],
                [t('Method'), detail.method],
                [t('Status'), detail.status_code],
                [t('Result'), t(auditResultLabel(detail.result))],
                [t('Username'), detail.username || t('Anonymous')],
                [t('User ID'), detail.user_id],
                [t('Path'), detail.path],
                [t('Route'), detail.route],
                [t('Query'), detail.query],
                [t('User Agent'), detail.user_agent],
              ].map(([label, value]) => (
                <div key={label} className='min-w-0'>
                  <dt className='text-[var(--semi-color-text-2)]'>{label}</dt>
                  <dd className='break-all'>
                    {value === '' || value == null ? '-' : value}
                  </dd>
                </div>
              ))}
            </dl>
            <BodyBlock
              label={t('Request Body')}
              value={detail.request_body}
              truncated={(detail.body_truncated & 1) !== 0}
            />
            <BodyBlock
              label={t('Response Body')}
              value={detail.response_body}
              truncated={(detail.body_truncated & 2) !== 0}
            />
          </>
        )}
      </div>
    </Modal>
  );
}
