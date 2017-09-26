
// Polyfill URL because Chrome and Firefox are not spec-compliant
// Hostnames of URIs with custom schemes (e.g. git) are not parsed out
import './util/polyfill'

import ErrorIcon from '@sourcegraph/icons/lib/Error'
import ServerIcon from '@sourcegraph/icons/lib/Server'
import * as React from 'react'
import { render } from 'react-dom'
import { BrowserRouter, Redirect, Route, RouteComponentProps, Switch } from 'react-router-dom'
import 'rxjs/add/observable/defer'
import 'rxjs/add/operator/catch'
import 'rxjs/add/operator/delay'
import 'rxjs/add/operator/do'
import 'rxjs/add/operator/retryWhen'
import 'rxjs/add/operator/switchMap'
import { Observable } from 'rxjs/Observable'
import { Subject } from 'rxjs/Subject'
import { Subscription } from 'rxjs/Subscription'
import { fetchCurrentUser } from './auth'
import { HeroPage } from './components/HeroPage'
import { Navbar } from './nav/Navbar'
import { ECLONEINPROGESS, EREPONOTFOUND, resolveRev } from './repo/backend'
import { Repository, RepositoryCloneInProgress, RepositoryNotFound } from './repo/Repository'
import { parseSearchURLQuery } from './search/index'
import { Search } from './search/Search'
import { SearchResults } from './search/SearchResults'
import { PasswordResetPage } from './settings/auth/PasswordResetPage'
import { SignInPage } from './settings/auth/SignInPage'
import { SettingsPage } from './settings/SettingsPage'
import { handleQueryEvents } from './tracking/analyticsUtils'
import { viewEvents } from './tracking/events'
import { ParsedRouteProps, parseRouteProps } from './util/routes'
import { sourcegraphContext } from './util/sourcegraphContext'
import { parseHash } from './util/url'

interface WithResolvedRevProps {
    component: any
    cloningComponent?: any
    notFoundComponent?: any // for 404s
    repoPath?: string
    rev?: string
    [key: string]: any
}

interface WithResolvedRevState {
    commitID?: string
    cloneInProgress: boolean
    notFound: boolean
}

class WithResolvedRev extends React.Component<WithResolvedRevProps, WithResolvedRevState> {
    public state: WithResolvedRevState = { cloneInProgress: false, notFound: false }
    private componentUpdates = new Subject<WithResolvedRevProps>()
    private subscriptions = new Subscription()

    constructor(props: WithResolvedRevProps) {
        super(props)
        this.subscriptions.add(
            this.componentUpdates
                .switchMap(({ repoPath, rev }) => {
                    if (!repoPath) {
                        return [undefined]
                    }
                    // Defer Observable so it retries the request on resubscription
                    return Observable.defer(() => resolveRev({ repoPath, rev }))
                        // On a CloneInProgress error, retry after 5s
                        .retryWhen(errors => errors
                            .do(err => {
                                if (err.code === ECLONEINPROGESS) {
                                    // Display cloning screen to the user and retry
                                    this.setState({ cloneInProgress: true })
                                    return
                                }
                                if (err.code === EREPONOTFOUND) {
                                    // Display 404to the user and do not retry
                                    this.setState({ notFound: true })
                                }
                                // Don't retry other errors
                                throw err
                            })
                            .delay(1000)
                        )
                        // Log other errors but don't break the stream
                        .catch(err => {
                            console.error(err)
                            return []
                        })
                })
                .subscribe(commitID => {
                    this.setState({ commitID, cloneInProgress: false })
                }, err => console.error(err))
        )
    }

    public componentDidMount(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillReceiveProps(nextProps: WithResolvedRevProps): void {
        if (this.props.repoPath !== nextProps.repoPath || this.props.rev !== nextProps.rev) {
            // clear state so the child won't render until the revision is resolved for new props
            this.state = { cloneInProgress: false, notFound: false }
            this.componentUpdates.next(nextProps)
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        if (this.props.notFoundComponent && this.state.notFound) {
            return <this.props.notFoundComponent {...this.props} />
        }
        if (this.props.cloningComponent && this.state.cloneInProgress) {
            return <this.props.cloningComponent {...this.props} />
        }
        if (this.props.repoPath && !this.state.commitID) {
            // commit not yet resolved but required if repoPath prop is provided;
            // render empty until commit resolved
            return null
        }
        return <this.props.component {...this.props} commitID={this.state.commitID} />
    }
}

class AppRouter extends React.Component<ParsedRouteProps, {}> {
    public componentDidMount(): void {
        this.logPageView(this.props)
    }

    public componentWillReceiveProps(nextProps: ParsedRouteProps): void {
        const thisHash = parseHash(nextProps.location.hash)
        const nextHash = parseHash(nextProps.location.hash)
        if (this.props.location.pathname !== nextProps.location.pathname ||
            this.props.location.search !== nextProps.location.search ||
            thisHash.modal !== nextHash.modal) {
            // Skip logging page view when only line/character is updated.
            this.logPageView(nextProps)
        }
    }

    public render(): JSX.Element | null {
        switch (this.props.routeName) {
            case 'search':
                return <SearchResults {...this.props} />

            case 'sign-in':
                return <SignInPage />

            case 'editor-auth':
            case 'settings-error':
            case 'team-profile':
            case 'teams-new':
            case 'user-profile':
            case 'accept-invite':
                // if on-prem, never show a settings page
                if (sourcegraphContext.onPrem) {
                    return <Redirect to='/search' />
                }
                return <SettingsPage {...this.props} />
            case 'password-reset':
                return <PasswordResetPage />
            case 'repository':
                return <WithResolvedRev {...this.props} component={Repository} cloningComponent={RepositoryCloneInProgress} notFoundComponent={RepositoryNotFound} />

            default:
                return <WithResolvedRev {...this.props} component={RepositoryNotFound} cloningComponent={RepositoryCloneInProgress} notFoundComponent={RepositoryNotFound} />
        }
    }

    private logPageView(props: ParsedRouteProps): void {
        const nextHash = parseHash(props.location.hash)
        switch (props.routeName) {
            case 'search':
                return viewEvents.SearchResults.log()
            case 'user-profile':
                return viewEvents.UserProfile.log()
            case 'editor-auth':
                return viewEvents.EditorAuth.log()
            case 'sign-in':
                return viewEvents.SignIn.log()
            case 'repository':
                return viewEvents.Blob.log({ fileShown: Boolean(props.filePath), referencesShown: nextHash.modal === 'references' })
        }
    }
}

/**
 * Defines the layout of all pages that have a navbar
 */
class Layout extends React.Component<RouteComponentProps<string[]>, {}> {
    public render(): JSX.Element | null {
        const props = parseRouteProps(this.props)
        return (
            <div className='layout'>
                <WithResolvedRev {...props} component={Navbar} cloningComponent={Navbar} notFoundComponent={Navbar} />
                <div className='layout__app-router-container'>
                    <AppRouter {...props} />
                </div>
            </div>
        )
    }
}

interface AppState {
    error?: Error
}

/**
 * handles rendering Search or SearchResults components based on whether or not
 * the search query (e.g. '?q=foo') is in URL.
 */
class SearchRouter extends React.Component<ParsedRouteProps, {}> {
    public componentDidMount(): void {
        this.logPageView(this.props)
    }

    public componentWillReceiveProps(nextProps: ParsedRouteProps): void {
        if (this.props.location.search !== nextProps.location.search) {
            this.logPageView(nextProps)
        }
    }

    public render(): JSX.Element | null {
        const searchOptions = parseSearchURLQuery(this.props.location.search)
        if (searchOptions.query) {
            return <Layout {...this.props} />
        }
        return <Search {...this.props} />
    }

    private logPageView(props: ParsedRouteProps): void {
        const searchOptions = parseSearchURLQuery(props.location.search)
        if (!searchOptions.query) {
            return viewEvents.Home.log()
        }
        // Other page views are logged by `Layout`.
    }
}

/**
 * The root component
 */
class App extends React.Component<{}, AppState> {

    constructor(props: {}) {
        super(props)
        this.state = {}
        // Fetch current user data
        fetchCurrentUser().subscribe(undefined, error => {
            console.error(error)
            this.setState({ error })
        })
    }

    public render(): JSX.Element | null {

        if (this.state.error) {
            return <HeroPage icon={ErrorIcon} title={'Something happened'} subtitle={this.state.error.message} />
        }

        if (window.pageError && window.pageError.statusCode !== 404) {
            const statusText = window.pageError.statusText
            const errorMessage = window.pageError.error
            const errorID = window.pageError.errorID

            let subtitle: JSX.Element | undefined
            if (errorID) {
                subtitle = (
                    <p>Sorry, there's been a problem. Please <a href='mailto:support@sourcegraph.com'>contact us</a> and include the error ID:
                        <span className='error-id'>{errorID}</span>
                    </p>
                )
            }
            if (errorMessage) {
                subtitle = (
                    <div className='app__error'>
                        {subtitle}
                        {subtitle && <hr />}
                        <pre>{errorMessage}</pre>
                    </div>
                )
            } else {
                subtitle = <div className='app__error'>{subtitle}</div>
            }
            return <HeroPage icon={ServerIcon} title={'500: ' + statusText} subtitle={subtitle} />
        }

        return (
            <BrowserRouter>
                <Switch>
                    <Route exact={true} path='/search' component={SearchRouter} />
                    <Route path='/*' component={Layout} />
                </Switch>
            </BrowserRouter>
        )
    }
}

window.addEventListener('DOMContentLoaded', () => {
    render(<App />, document.querySelector('#root'))
})

handleQueryEvents(window.location.href)
