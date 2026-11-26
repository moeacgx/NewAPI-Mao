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
import { Tabs, Typography } from '@douyinfe/semi-ui';
import { Briefcase } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import SupplierApplicationForm from '../../components/supplier/SupplierApplicationForm';
import MySupplierSubmissions from '../../components/supplier/MySupplierSubmissions';

const { Title } = Typography;

const SupplierApplication = () => {
  const { t } = useTranslation();

  return (
    <main className='classic-console-page'>
      <div className='classic-console-page-container'>
        <section className='classic-console-panel'>
          <div className='classic-console-panel-header flex items-center gap-2 px-4 py-3 text-blue-500'>
            <Briefcase size={20} />
            <Title heading={3} className='!mb-0 !font-bold'>
              {t('Supplier Application')}
            </Title>
          </div>
          <div className='classic-console-panel-content'>
            <Tabs
              type='line'
              className='supplier-application-tabs'
              defaultActiveKey='submit'
            >
              <Tabs.TabPane tab={t('Submit a quote')} itemKey='submit'>
                <SupplierApplicationForm />
              </Tabs.TabPane>
              <Tabs.TabPane tab={t('My submissions')} itemKey='my-submissions'>
                <MySupplierSubmissions />
              </Tabs.TabPane>
            </Tabs>
          </div>
        </section>
      </div>
    </main>
  );
};

export default SupplierApplication;
