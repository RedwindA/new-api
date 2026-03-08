import React from 'react';
import { Tooltip } from '@douyinfe/semi-ui';
import { IconInfoCircle } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const tooltipStyle = { maxWidth: 'min(300px, 90vw)', wordBreak: 'break-word' };
const iconStyle = { cursor: 'pointer', color: 'var(--semi-color-text-2)' };

export default function SearchHelpTooltip({ size }) {
  const { t } = useTranslation();
  return (
    <Tooltip
      content={t('搜索支持多关键词（空格分隔，全部匹配），使用 -前缀排除，例如：gpt -mini -realtime')}
      position='bottomRight'
      style={tooltipStyle}
    >
      <IconInfoCircle style={iconStyle} size={size} />
    </Tooltip>
  );
}
