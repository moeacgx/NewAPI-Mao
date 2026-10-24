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
export const ticketStatuses = {
  open: 'Open',
  processing: 'In progress',
  resolved: 'Resolved',
  closed: 'Closed',
  archived: 'Archived',
};
export const ticketCategories = {
  general: 'General inquiry',
  billing: 'Billing',
  technical: 'Technical issue',
  account: 'Account',
};
export const ticketPriorities = { low: 'Low', normal: 'Normal', high: 'High' };
export const ticketStatusColors = {
  open: 'blue',
  processing: 'orange',
  resolved: 'green',
  closed: 'grey',
  archived: 'grey',
};

export function ticketError(error, fallback) {
  return error?.response?.data?.message || error?.message || fallback;
}

export function ticketResponse(response, fallback) {
  if (!response?.data?.success)
    throw new Error(response?.data?.message || fallback);
  return response.data.data;
}

export function ticketTime(timestamp) {
  if (!timestamp) return '—';
  return new Date(timestamp * 1000).toLocaleString();
}
