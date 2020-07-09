import { storiesOf } from '@storybook/react'
import { radios } from '@storybook/addon-knobs'
import React from 'react'
import { DismissibleAlert } from './DismissibleAlert'
import webStyles from '../SourcegraphWebApp.scss'
import { MemoryRouter } from 'react-router'

const { add } = storiesOf('web/DismissibleAlert', module).addDecorator(story => {
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

add('Info alert', () => (
    <DismissibleAlert className="alert-info" partialStorageKey="test-key-123">
        Some content that is dismissible.
    </DismissibleAlert>
))
