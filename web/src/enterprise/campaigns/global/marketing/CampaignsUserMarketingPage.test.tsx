import React from 'react'
import renderer from 'react-test-renderer'
import { CampaignsUserMarketingPage } from './CampaignsUserMarketingPage'

describe('CampaignsUserMarketingPage', () => {
    test('renders', () => {
        const result = renderer.create(<CampaignsUserMarketingPage />)
        expect(result.toJSON()).toMatchSnapshot()
    })
})
