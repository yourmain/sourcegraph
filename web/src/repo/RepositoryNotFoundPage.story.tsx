import { storiesOf } from '@storybook/react'
import { radios, boolean } from '@storybook/addon-knobs'
import React from 'react'
import { RepositoryNotFoundPage } from './RepositoryNotFoundPage'
import webStyles from '../SourcegraphWebApp.scss'
import { MemoryRouter } from 'react-router'

const { add } = storiesOf('web/repo/RepositoryNotFoundPage', module).addDecorator(story => {
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

add('Reference tag', () => (
    <RepositoryNotFoundPage
        repo="github.com/sourcegraph/aewsome"
        viewerCanAdminister={boolean('viewerCanAdminister', true)}
    />
))
