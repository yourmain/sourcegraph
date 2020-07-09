import { createMemoryHistory } from 'history'
import { storiesOf } from '@storybook/react'
import { radios } from '@storybook/addon-knobs'
import React from 'react'
import { RepositoryReleasesTagsPage } from './RepositoryReleasesTagsPage'
import webStyles from '../../SourcegraphWebApp.scss'
import { MemoryRouter } from 'react-router'
import { of } from 'rxjs'

const { add } = storiesOf('web/repo/releases/RepositoryReleasesTagsPage', module).addDecorator(story => {
    // TODO find a way to do this globally for all stories and storybook itself.
    const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
    document.body.classList.toggle('theme-light', theme === 'light')
    document.body.classList.toggle('theme-dark', theme === 'dark')
    return (
        <MemoryRouter>
            <style>{webStyles}</style>
            <div className="p-3 container">{story()}</div>
        </MemoryRouter>
    )
})

add('List tags', () => {
    const history = createMemoryHistory()
    return (
        <RepositoryReleasesTagsPage
            history={history}
            location={history.location}
            repo={{ id: '123' }}
            queryGitReferences={() =>
                of({
                    totalCount: 1,
                    nodes: [
                        {
                            id: 'id123',
                            displayName: '3.18',
                            target: {
                                commit: {
                                    author: { date: new Date().toISOString(), person: { displayName: 'John Doe' } },
                                    committer: null,
                                    behindAhead: {
                                        ahead: 10,
                                        behind: 20,
                                        __typename: 'BehindAheadCounts',
                                    },
                                },
                            },
                            url: 'http://test.test/3.18',
                        },
                    ],
                    __typename: 'GitRefConnection',
                    pageInfo: { __typename: 'PageInfo', endCursor: null, hasNextPage: false },
                })
            }
        />
    )
})
