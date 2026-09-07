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

import React, { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Banner, Button, Empty, Tag, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers';
import CardTable from '../../components/common/ui/CardTable';
import CardPro from '../../components/common/ui/CardPro';
import AuditRequestFilters, { auditTextFilters } from './filters';
import AuditRequestDetail from './detail';
import { auditResultLabel } from './result';

export default function AuditRequestLogs() {
  const { t } = useTranslation();
  const [search, setSearch] = useSearchParams();
  const [state, setState] = useState({ items: [], total: 0, loading: true });
  const [statusCodes, setStatusCodes] = useState([]);
  const [selected, setSelected] = useState(null);
  const [retry, setRetry] = useState(0);
  const searchString = search.toString();
  const filters = useMemo(
    () => Object.fromEntries(new URLSearchParams(searchString)),
    [searchString],
  );
  const parsedPage = Number(filters.p);
  const page =
    Number.isSafeInteger(parsedPage) && parsedPage > 0 ? parsedPage : 1;
  const pageSize = [10, 20, 50, 100].includes(Number(filters.page_size))
    ? Number(filters.page_size)
    : 20;
  const query = new URLSearchParams({
    ...filters,
    p: String(page),
    page_size: String(pageSize),
  }).toString();

  useEffect(() => {
    const controller = new AbortController();
    API.get('/api/audit_request/status_codes', {
      signal: controller.signal,
      disableDuplicate: true,
    })
      .then(({ data }) => {
        if (!controller.signal.aborted && data.success)
          setStatusCodes(data.data || []);
      })
      .catch(() => {});
    return () => controller.abort();
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setState({ items: [], total: 0, loading: true });
    API.get(`/api/audit_request/?${query}`, {
      signal: controller.signal,
      disableDuplicate: true,
    })
      .then(({ data }) => {
        if (controller.signal.aborted) return;
        if (!data.success) {
          const disabled = data.message === 'Request audit is not enabled';
          setState({
            items: [],
            total: 0,
            disabled,
            error: disabled
              ? ''
              : data.message || t('Failed to load audit request logs'),
          });
          return;
        }
        setState({
          items: data.data?.items || [],
          total: data.data?.total || 0,
        });
      })
      .catch(() => {
        if (!controller.signal.aborted)
          setState({
            items: [],
            total: 0,
            error: t('Failed to load audit request logs'),
          });
      });
    return () => controller.abort();
  }, [query, retry, t]);

  const columns = [
    {
      title: t('Time'),
      dataIndex: 'created_at',
      render: (value) => new Date(value * 1000).toLocaleString(),
    },
    { title: t('IP'), dataIndex: 'ip' },
    { title: t('Method'), dataIndex: 'method' },
    {
      title: t('Path'),
      dataIndex: 'path',
      render: (value) => <span className='break-all'>{value}</span>,
    },
    { title: t('Status'), dataIndex: 'status_code' },
    {
      title: t('Result'),
      dataIndex: 'result',
      render: (value) => {
        let color = 'grey';
        if (value === 1) color = 'green';
        if (value === 2) color = 'red';
        return <Tag color={color}>{t(auditResultLabel(value))}</Tag>;
      },
    },
    {
      title: t('Latency'),
      dataIndex: 'latency_ms',
      render: (value) => `${value} ms`,
    },
    {
      title: t('Username'),
      dataIndex: 'username',
      render: (value) => value || t('Anonymous'),
    },
    {
      title: t('Request ID'),
      dataIndex: 'request_id',
      render: (value) => <span className='break-all'>{value}</span>,
    },
    {
      title: t('Request details'),
      key: 'details',
      render: (_, log) => (
        <Button onClick={() => setSelected(log)}>
          {t('View request details')}
        </Button>
      ),
    },
  ];

  return (
    <div className='mt-[60px] px-2 pb-4'>
      <CardPro
        type='type2'
        t={t}
        statsArea={
          <Typography.Title heading={4}>
            {t('Audit Request Logs')}
          </Typography.Title>
        }
        searchArea={
          <AuditRequestFilters
            filters={filters}
            statusCodes={statusCodes}
            onApply={(draft) => {
              const next = new URLSearchParams({
                p: '1',
                page_size: String(pageSize),
              });
              for (const key of [
                ...auditTextFilters.map(([key]) => key),
                'status_code',
                'result',
                'start_timestamp',
                'end_timestamp',
              ]) {
                const value = String(draft[key] ?? '').trim();
                if (value) next.set(key, value);
              }
              setSearch(next);
            }}
          />
        }
      >
        {state.error && (
          <div className='mb-3 space-y-2'>
            <Banner
              type='danger'
              description={state.error}
              closeIcon={null}
              fullMode={false}
            />
            <Button onClick={() => setRetry((value) => value + 1)}>
              {t('Retry')}
            </Button>
          </div>
        )}
        <CardTable
          columns={columns}
          dataSource={state.items}
          loading={Boolean(state.loading)}
          rowKey={(log) => `${log.request_id}:${log.created_at}`}
          scroll={{ x: 1400 }}
          empty={
            <Empty
              title={
                state.disabled
                  ? t('Request audit is not enabled')
                  : t('No audit request logs')
              }
              description={
                state.disabled
                  ? t('Set AUDIT_SQL_DSN to enable request audit logging.')
                  : t('No requests match the current filters.')
              }
            />
          }
          pagination={{
            currentPage: page,
            pageSize,
            total: state.total,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            onPageChange: (nextPage) =>
              setSearch({
                ...filters,
                p: String(nextPage),
                page_size: String(pageSize),
              }),
            onPageSizeChange: (size) =>
              setSearch({ ...filters, p: '1', page_size: String(size) }),
          }}
        />
      </CardPro>
      {selected && (
        <AuditRequestDetail log={selected} onClose={() => setSelected(null)} />
      )}
    </div>
  );
}
