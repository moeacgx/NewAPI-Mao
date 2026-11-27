/*
Copyright (C) 2023-2026 QuantumNous

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

import React from 'react';
import { Typography } from '@douyinfe/semi-ui';
import { Briefcase } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import SupplierApplicationManagementTable from '../../components/supplier/SupplierApplicationManagementTable';

const { Title } = Typography;

const SupplierApplicationManagement = () => {
  const { t } = useTranslation();

  return (
    <main className='classic-console-page'>
      <div className='classic-console-page-container'>
        <section className='classic-console-panel'>
          <div className='classic-console-panel-header flex items-center gap-2 px-4 py-3 text-blue-500'>
            <Briefcase size={20} />
            <Title heading={3} className='!mb-0 !font-bold'>
              {t('Supplier Application Management')}
            </Title>
          </div>
          <div className='classic-console-panel-content'>
            <SupplierApplicationManagementTable />
          </div>
        </section>
      </div>
    </main>
  );
};

export default SupplierApplicationManagement;
