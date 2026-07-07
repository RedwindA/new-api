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

export const ADMIN_PERMISSION_RESOURCES = {
  CHANNEL: 'channel',
};

export const ADMIN_PERMISSION_ACTIONS = {
  SENSITIVE_WRITE: 'sensitive_write',
};

const ROLE_ROOT = 100;

function getCurrentUserRole() {
  try {
    const user = JSON.parse(localStorage.getItem('user'));
    return typeof user?.role === 'number' ? user.role : 0;
  } catch (error) {
    return 0;
  }
}

export function hasAdminPermission(permissions, resource, action) {
  if (!permissions) {
    // 权限尚未从后端加载时回退到本地缓存的角色，避免 root 用户在加载期间被锁定；
    // 加载完成后以后端返回的 admin_permissions 为准（root 在后端已解析为全部允许）
    return getCurrentUserRole() === ROLE_ROOT;
  }
  return permissions.admin_permissions?.[resource]?.[action] === true;
}
