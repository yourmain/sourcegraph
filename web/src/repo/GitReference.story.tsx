import { storiesOf } from '@storybook/react'
import { radios, boolean } from '@storybook/addon-knobs'
import React from 'react'
import { GitReferenceNode } from './GitReference'
import webStyles from '../SourcegraphWebApp.scss'
import { MemoryRouter } from 'react-router'

const { add } = storiesOf('web/repo/GitReference', module).addDecorator(story => {
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

add('Branch 3.18', () => (
    <GitReferenceNode
        node={{
            displayName: '3.18',
            id: 'gitrefid',
            url: 'http://test.test/3.18',
            target: {
                commit: {
                    author: { date: new Date().toISOString(), person: { displayName: 'John Doe' } },
                    committer: null,
                    behindAhead: { __typename: 'BehindAheadCounts', ahead: 10, behind: 5 },
                },
            },
        }}
        ancestorIsLink={boolean('ancestorIsLink', false)}
    />
))
