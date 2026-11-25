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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout/components/section-page-layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { SupplierApplicationForm } from './application-form'
import { MySupplierSubmissions } from './my-submissions'

export function SupplierApplications() {
  const { t } = useTranslation()
  const [tab, setTab] = useState('submit')

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Supplier Application')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle>{t('How it works')}</CardTitle>
              <CardDescription>
                {t(
                  'Tell us your site and the groups you can supply with a ratio for each. We compare quotes across sites and contact you through the details you leave when there is a fit.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <ol className='grid gap-2 text-sm sm:grid-cols-3'>
                <li className='bg-muted/20 rounded-lg border p-3'>
                  <span className='font-medium'>1. {t('Submit a quote')}</span>
                </li>
                <li className='bg-muted/20 rounded-lg border p-3'>
                  <span className='font-medium'>
                    2. {t('Platform compares')}
                  </span>
                </li>
                <li className='bg-muted/20 rounded-lg border p-3'>
                  <span className='font-medium'>3. {t('We get in touch')}</span>
                </li>
              </ol>
              <Alert variant='destructive'>
                <AlertTitle>{t('Do not enter API keys here')}</AlertTitle>
                <AlertDescription>
                  {t(
                    'This form never needs API keys, passwords, or any access credentials. Submissions cannot be edited after sending, so confirm everything before you submit.'
                  )}
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
          <Card>
            <CardContent>
              <Tabs value={tab} onValueChange={setTab} className='space-y-6'>
                <TabsList className='max-w-full flex-wrap justify-start group-data-horizontal/tabs:h-auto'>
                  <TabsTrigger value='submit'>
                    {t('Submit application')}
                  </TabsTrigger>
                  <TabsTrigger value='mine'>{t('My submissions')}</TabsTrigger>
                </TabsList>
                {/* Keep the form mounted so a half-filled draft survives tab switches. */}
                <TabsContent value='submit' keepMounted>
                  <SupplierApplicationForm onSubmitted={() => setTab('mine')} />
                </TabsContent>
                <TabsContent value='mine'>
                  <MySupplierSubmissions />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
