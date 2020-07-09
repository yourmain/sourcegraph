import { createMemoryHistory } from 'history'
import { storiesOf } from '@storybook/react'
import { radios } from '@storybook/addon-knobs'
import React from 'react'
import { RepositoryCommitPage } from './RepositoryCommitPage'
import webStyles from '../../SourcegraphWebApp.scss'
import { MemoryRouter } from 'react-router'
import { of } from 'rxjs'

const { add } = storiesOf('web/repo/commit/RepositoryCommitPage', module).addDecorator(story => {
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

add('Show commit', () => {
    const history = createMemoryHistory()
    return (
        <RepositoryCommitPage
            history={history}
            location={history.location}
            extensionsController={{}}
            isLightTheme={true}
            telemetryService={{}}
            platformContext={{}}
            onDidUpdateExternalLinks={() => undefined}
            repo={{
                name: '123123',
            }}
            match={{ isExact: true, params: { revspec: 'asd' }, path: 'asd', url: 'sdkask' }}
            _queryCommit={() =>
                of({
                    __typename: 'GitCommit',
                    subject: 'Test subject',
                    canonicalURL: 'http://test.test/commit',
                    message: 'Super commit message.',
                    body: 'Body',
                    id: 'id123',
                    abbreviatedOID: '123def',
                    oid: '123defdef123',
                    tree: {
                        canonicalURL: 'http://test.test/tree',
                    },
                    author: {
                        date: new Date().toISOString(),
                        person: {
                            name: 'name',
                        },
                    },
                    committer: null,
                    parents: [{ oid: 'abcdef', abbreviatedOID: 'abcdef1', url: 'http://test.test/parent' }],
                })
            }
        />
    )
})
