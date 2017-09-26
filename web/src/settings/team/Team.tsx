import CloseIcon from '@sourcegraph/icons/lib/Close'
import * as React from 'react'
import { Redirect } from 'react-router'
import reactive from 'rx-component'
import 'rxjs/add/observable/merge'
import 'rxjs/add/operator/combineLatest'
import 'rxjs/add/operator/combineLatest'
import 'rxjs/add/operator/distinctUntilChanged'
import 'rxjs/add/operator/filter'
import 'rxjs/add/operator/map'
import 'rxjs/add/operator/mergeMap'
import 'rxjs/add/operator/scan'
import { Observable } from 'rxjs/Observable'
import { Subject } from 'rxjs/Subject'
import { currentUser } from '../../auth'
import { removeUserFromOrg } from '../backend'
import { UserAvatar } from '../user/UserAvatar'
import { InviteForm } from './InviteForm'

export interface Props {
    teamName: string
}

interface State {
    org?: GQL.IOrg
    user?: GQL.IUser
}

type Update = (s: State) => State

/**
 * The team settings page
 */
export const Team = reactive<Props>(props => {

    const memberRemoves = new Subject<GQL.IOrgMember>()

    return Observable.merge<Update>(
        props
            .map(props => props.teamName)
            .distinctUntilChanged()
            .combineLatest(currentUser)
            .map(([teamName, user]) => (state: State): State => ({
                ...state,
                org: user && user.orgs.find(org => org.name === teamName) || undefined,
                user: user || undefined
            })),
        memberRemoves
            .combineLatest(currentUser)
            .filter(([member, user]) => !!user && confirm(
                user.id === member.userID
                    ? `Leave this team?`
                    : `Remove ${member.displayName} from this team?`
            ))
            .mergeMap(([memberToRemove]) =>
                removeUserFromOrg(memberToRemove.org.id, memberToRemove.userID)
                    .map(() => (state: State): State => ({
                        ...state,
                        org: state.org && {
                            ...state.org,
                            members: state.org.members.filter(member => member.userID === memberToRemove.userID)
                        }
                    }))
            )
    )
        .scan<Update, State>((state: State, update: Update) => update(state), {})
        .map(({ user, org }: State): JSX.Element | null => {
            if (!user) {
                return <Redirect to='/sign-in' />
            }
            if (!org) {
                // TODO make prettier
                return <span>Team not found</span>
            }
            return (
                <div className='team'>
                    <h1>{org.name}</h1>

                    <InviteForm orgID={org.id}/>

                    <table className='table table-hover'>
                        <thead>
                            <tr>
                                <th></th>
                                <th>Name</th>
                                <th>Username</th>
                                <th>Email</th>
                                <th></th>
                            </tr>
                        </thead>
                        <tbody>
                            {
                                org.members.map(member => (
                                    <tr key={member.id}>
                                        <td className='team__avatar-cell'><UserAvatar user={member} size={64}/></td>
                                        <td>{member.displayName}</td>
                                        <td>{member.username}</td>
                                        <td>{member.email}</td>
                                        <td className='team__actions-cell'>
                                            <button
                                                className='btn btn-icon'
                                                title={user.id === member.userID ? 'Leave' : 'Remove'}
                                                onClick={() => memberRemoves.next({ ...member, org })}
                                            >
                                                <CloseIcon />
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            }
                        </tbody>
                    </table>
                </div>
            )
        })
})
