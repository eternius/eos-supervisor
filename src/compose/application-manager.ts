import * as _ from 'lodash';

import { Transaction } from '../db';
import { checkInt } from '../lib/validation';
import { ComposeNetworkConfig } from './types/network';
// FIXME: Change this type to ComposeServiceConfig
import { ServiceComposeConfig } from './types/service';
import { ComposeVolumeConfig } from './volume';
import Images from './images';

export interface ApplicationTargetState {
	[appId: string]: {
		// FIXME: Before merge, try this code with an empty
		// state endpoint
		name: string;
		commit: string;
		releaseId: number;
		services: {
			[serviceId: string]: Partial<ServiceComposeConfig>;
		};
		networks: Dictionary<Partial<ComposeNetworkConfig>>;
		volumes: Dictionary<Partial<ComposeVolumeConfig>>;
	};
}

export type DependentTargetState = Dictionary<unknown>;

// FIXME: Make this an event-emitter
export class ApplicationManager {
	private images: Images;

	public constructor() {
		// TODO
	}

	public setTarget(
		target: ApplicationTargetState,
		_dependent: DependentTargetState,
		source: any,
		_trx: Nullable<Transaction>,
	) {
		const setInTransaction = async (trx: Transaction) => {
			const appIds = _.keys(target);
			for (const strAppId of appIds) {
				const appId = checkInt(strAppId);
				if (appId == null) {
					throw new Error('Application without application ID!');
				}
				const clonedApp = {
					..._.clone(target[appId]),
					appId,
					source,
				};

				const normalised = await this.normalizeAppForDB(clonedApp);
			}
		};
	}

	private async normalizeAppForDB(
		app: ApplicationTargetState[''] & { appId: number; source: string },
	) {
		const services = await Promise.all(
			_.map(app.services, async (svc, serviceId) => {
				if (!svc.image) {
					throw new Error(
						'service.image must be defined when storing an app to the database',
					);
				}
				const image = this.images.normalise(svc.image);
				return {
					...svc,
					appId: app.appId,
					releaseId: app.releaseId,
					serviceId: checkInt(serviceId),
					commit: app.commit,
					image,
				};
			}),
		);

		return {
			appId: app.appId,
			commit: app.commit,
			name: app.name,
			source: app.source,
			releaseId: app.releaseId,
			services: JSON.stringify(services),
			networks: JSON.stringify(app.networks || {})
			volumes: JSON.stringify(app.volumes)
		}
	}
}

export default ApplicationManager;
