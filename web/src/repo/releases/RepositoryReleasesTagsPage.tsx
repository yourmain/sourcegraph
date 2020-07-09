import React, { useEffect, useCallback } from 'react'
import * as H from 'history'
import * as GQL from '../../../../shared/src/graphql/schema'
import { FilteredConnection, FilteredConnectionQueryArgs, Connection } from '../../components/FilteredConnection'
import { PageTitle } from '../../components/PageTitle'
import { eventLogger } from '../../tracking/eventLogger'
import {
    GitReferenceNode,
    queryGitReferences as queryGitReferencesFromBackend,
    GitReferenceType,
} from '../GitReference'
import { RepositoryReleasesAreaPageProps } from './RepositoryReleasesArea'
import { Observable } from 'rxjs'

interface Props extends RepositoryReleasesAreaPageProps {
    history: H.History
    location: H.Location
    queryGitReferences?: (args: {
        repo: GQL.ID
        first?: number
        query?: string
        type: GQL.GitRefType
        withBehindAhead?: boolean
    }) => Observable<Connection<GitReferenceType>>
}

/** A page that shows all of a repository's tags. */
export const RepositoryReleasesTagsPage: React.FunctionComponent<Props> = ({
    repo,
    history,
    location,
    queryGitReferences: queryGitReferences = queryGitReferencesFromBackend,
}) => {
    useEffect(() => {
        eventLogger.logViewEvent('RepositoryReleasesTags')
    }, [])

    const queryTags = useCallback(
        (args: FilteredConnectionQueryArgs): Observable<Connection<GitReferenceType>> =>
            queryGitReferences({ ...args, repo: repo.id, type: GQL.GitRefType.GIT_TAG }),
        [repo.id, queryGitReferences]
    )

    return (
        <div className="repository-releases-page">
            <PageTitle title="Tags" />
            <FilteredConnection<GitReferenceType>
                className="my-3"
                listClassName="list-group list-group-flush"
                noun="tag"
                pluralNoun="tags"
                queryConnection={queryTags}
                nodeComponent={GitReferenceNode}
                defaultFirst={20}
                autoFocus={true}
                history={history}
                location={location}
            />
        </div>
    )
}
