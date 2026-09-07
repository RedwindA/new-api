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
import { Button, DatePicker, Input, Select } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { DATE_RANGE_PRESETS } from '../../constants/console.constants';

export const auditTextFilters = [
  ['ip', 'IP'],
  ['path', 'Path'],
  ['method', 'Method'],
  ['username', 'Username'],
  ['user_id', 'User ID'],
  ['request_id', 'Request ID'],
];

export default function AuditRequestFilters(props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(props.filters);

  useEffect(() => {
    setDraft(props.filters);
  }, [props.filters]);

  return (
    <form
      className='mb-4 space-y-3'
      onSubmit={(event) => {
        event.preventDefault();
        props.onApply(draft);
      }}
    >
      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4'>
        {auditTextFilters.map(([key, label]) => (
          <label key={key} className='min-w-0 space-y-1'>
            <span className='text-sm'>{t(label)}</span>
            <Input
              value={draft[key] || ''}
              aria-label={t(label)}
              onChange={(value) =>
                setDraft((current) => ({ ...current, [key]: value }))
              }
              autoComplete='off'
            />
          </label>
        ))}
        <label className='min-w-0 space-y-1'>
          <span className='text-sm'>{t('Status')}</span>
          <Select
            multiple
            className='w-full'
            aria-label={t('Status')}
            placeholder={t('Status')}
            value={
              draft.status_code ? draft.status_code.split(',').map(Number) : []
            }
            optionList={props.statusCodes.map((code) => ({
              label: String(code),
              value: code,
            }))}
            onChange={(values) =>
              setDraft((current) => ({
                ...current,
                status_code: values.join(','),
              }))
            }
          />
        </label>
        <label className='min-w-0 space-y-1'>
          <span className='text-sm'>{t('Result')}</span>
          <Select
            className='w-full'
            aria-label={t('Result')}
            value={draft.result || ''}
            optionList={[
              { label: t('All results'), value: '' },
              { label: t('Failed'), value: '2' },
              { label: t('Success'), value: '1' },
            ]}
            onChange={(value) =>
              setDraft((current) => ({ ...current, result: value }))
            }
          />
        </label>
        <div className='min-w-0 sm:col-span-2'>
          <DatePicker
            className='w-full'
            type='dateTimeRange'
            placeholder={[t('Start Time'), t('End Time')]}
            showClear
            size='small'
            value={['start_timestamp', 'end_timestamp'].map((key) => {
              const date = draft[key]
                ? new Date(Number(draft[key]) * 1000)
                : null;
              return date && Number.isFinite(date.getTime()) ? date : undefined;
            })}
            presets={DATE_RANGE_PRESETS.map((preset) => ({
              text: t(preset.text),
              start: preset.start(),
              end: preset.end(),
            }))}
            onChange={(_, dates) => {
              setDraft((current) => ({
                ...current,
                start_timestamp: dates?.[0]
                  ? String(Math.floor(dates[0].getTime() / 1000))
                  : '',
                end_timestamp: dates?.[1]
                  ? String(Math.floor(dates[1].getTime() / 1000))
                  : '',
              }));
            }}
          />
        </div>
      </div>
      <div className='flex justify-end gap-2'>
        <Button
          onClick={() => {
            setDraft({});
            props.onApply({});
          }}
        >
          {t('Reset')}
        </Button>
        <Button htmlType='submit' theme='solid'>
          {t('Search')}
        </Button>
      </div>
    </form>
  );
}
