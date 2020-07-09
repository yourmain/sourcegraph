import { storiesOf } from '@storybook/react'
import { radios } from '@storybook/addon-knobs'
import React from 'react'
import { ExternalServiceCard } from './ExternalServiceCard'
import webStyles from '../SourcegraphWebApp.scss'
import { ExternalServiceKind } from '../../../shared/src/graphql/schema'
import { MemoryRouter } from 'react-router'
import { defaultExternalServices } from '../site-admin/externalServices'

const { add } = storiesOf('web/ExternalServiceCard', module).addDecorator(story => {
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

add('Overview', () => (
    <>
        {Object.values(ExternalServiceKind).map((kind, index) => (
            <ExternalServiceCard
                key={index}
                {...defaultExternalServices[kind]}
                kind={kind}
                to="/there"
                title="Awesome service"
                shortDescription="Short descriptive text."
            />
        ))}
    </>
))
