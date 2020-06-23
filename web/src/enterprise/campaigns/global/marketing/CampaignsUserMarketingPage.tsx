import React from 'react'
import { CampaignsMarketing } from './CampaignsMarketing'

export const CampaignsUserMarketingPage: React.FunctionComponent<{}> = () => (
    <CampaignsMarketing
        body={
            <section className="my-3">
                <h2>Interested in campaigns?</h2>
                <p>
                    At this time, creating and managing campaigns is only available to Sourcegraph admins. What can you
                    do?
                </p>
                <div className="row">
                    <ol>
                        <li>Let your Sourcegraph admin know you're interested in using campaigns for your team.</li>
                        <li>
                            Learn how to{' '}
                            <a href="https://docs.sourcegraph.com/user/campaigns#creating-campaigns">
                                get started creating campaigns
                            </a>
                            .
                        </li>
                    </ol>
                </div>

                <a href="https://docs.sourcegraph.com/user/campaigns" rel="noopener" className="btn btn-primary">
                    Read more about campaigns
                </a>
            </section>
        }
    />
)
