// Every code sample on the site, cut from the files CI compiles and tests. Pages spread one into
// Starlight's <Code>: <Code {...userAggregate} />. Never paste code into a page instead.
import { goDecls, yamlBlock } from './snippet';

import aggregateGo from '../../../internal/platform/es/aggregate.go?raw';
import apperrGo from '../../../internal/platform/apperr/apperr.go?raw';
import authClientsGo from '../../../internal/auth/app/clients.go?raw';
import authMysqlGo from '../../../internal/auth/store/mysql/mysql.go?raw';
import bufGenYaml from '../../../buf.gen.yaml?raw';
import bufYaml from '../../../buf.yaml?raw';
import esGo from '../../../internal/platform/es/es.go?raw';
import esTestGo from '../../../internal/platform/es/es_test.go?raw';
import estestStoreGo from '../../../internal/platform/es/estest/store.go?raw';
import loginProto from '../../../proto/user/v1/login.proto?raw';
import memdbTestGo from '../../../internal/platform/memdb/memdb_test.go?raw';
import outboxGo from '../../../internal/platform/outbox/outbox.go?raw';
import registryGo from '../../../internal/platform/es/registry.go?raw';
import relayGo from '../../../internal/platform/outbox/relay.go?raw';
import relayTestGo from '../../../internal/platform/outbox/relay_test.go?raw';
import repositoryGo from '../../../internal/platform/es/repository.go?raw';
import scenarioGo from '../../../internal/platform/es/estest/scenario.go?raw';
import uowGo from '../../../internal/platform/uow/uow.go?raw';
import userAppGo from '../../../internal/user/app/app.go?raw';
import userEventsGo from '../../../internal/user/domain/events.go?raw';
import userGo from '../../../internal/user/domain/user.go?raw';
import userHttpapiGo from '../../../internal/user/httpapi/httpapi.go?raw';
import userMatrixTestGo from '../../../internal/user/httpapi/matrix_test.go?raw';
import userOpenAPI from '../../../api/user.openapi.yaml?raw';
import userTestGo from '../../../internal/user/domain/user_test.go?raw';

export interface Sample {
	code: string;
	lang: string;
	title: string;
}

function go(source: string, file: string, ...starts: string[]): Sample {
	return { code: goDecls(source, file, ...starts), lang: 'go', title: file };
}

const USER = 'internal/user/domain/user.go';
const USER_EVENTS = 'internal/user/domain/events.go';
const USER_TEST = 'internal/user/domain/user_test.go';
const USER_APP = 'internal/user/app/app.go';

export const userAggregate = go(
	userGo,
	USER,
	'type User struct',
	'func New()',
	'func (u *User) ID',
	'func (u *User) ChangeEmail(',
	'func (u *User) Apply(',
);
// The landing page shows only the decision and how it is applied, so the hero stays short on a phone.
export const userDecide = go(userGo, USER, 'func (u *User) ChangeEmail(', 'func (u *User) Apply(');
export const userEvents = go(userEventsGo, USER_EVENTS, 'type Registered struct', 'type EmailChanged struct', 'type GoogleLinked struct');
export const userEmailChanged = go(userEventsGo, USER_EVENTS, 'type EmailChanged struct');
export const userRegistry = go(userEventsGo, USER_EVENTS, 'func RegisterEvents(');
export const userChangeEmailTest = go(userTestGo, USER_TEST, 'func TestChangeEmail(');
export const userChangeEmailTests = go(userTestGo, USER_TEST, 'func TestChangeEmail(', 'func TestChangeEmailRules(');
export const userLinkGoogle = go(userAppGo, USER_APP, 'func (s *Service) LinkGoogle(');
export const userConfirmEmailChange = go(userAppGo, USER_APP, 'func (s *Service) ConfirmEmailChange(');
export const userRequestEmailChange = go(userAppGo, USER_APP, 'func (s *Service) RequestEmailChange(');
export const userActor = go(userAppGo, USER_APP, 'func Actor(');
export const userEmailChangeHandler = go(
	userHttpapiGo,
	'internal/user/httpapi/httpapi.go',
	'func (h *handler) requestEmailChange(',
);
export const userMatrixRows = go(userMatrixTestGo, 'internal/user/httpapi/matrix_test.go', 'var rows = []access{');

// The event-sourcing kit's own test aggregate, a counter, shows what the user aggregate has no short
// example of: a conflict, a rollback and an upcast.
const ES_TEST = 'internal/platform/es/es_test.go';

export const counterRepository = go(esTestGo, ES_TEST, 'func newRepository(');
export const counterScenario = go(esTestGo, ES_TEST, 'func TestScenario');
export const counterConflictTest = go(esTestGo, ES_TEST, 'func TestRepositoryConflict');
export const counterProjectionRollbackTest = go(esTestGo, ES_TEST, 'func TestRepositoryProjectionFailureRollsBack');
export const counterUpcastTest = go(esTestGo, ES_TEST, 'func TestUpcast');

export const eventEnvelope = go(esGo, 'internal/platform/es/es.go', 'type Event struct', 'type Meta struct');
export const eventStore = go(esGo, 'internal/platform/es/es.go', 'var ErrConcurrency', 'type Store interface');
export const aggregateBase = go(
	aggregateGo,
	'internal/platform/es/aggregate.go',
	'type Changes struct',
	'func Raise(',
	'func Replay(',
	'func Pending(',
);
export const registryApi = go(
	registryGo,
	'internal/platform/es/registry.go',
	'func Register[',
	'func (r *Registry) Upcast(',
);
export const projectionApi = go(
	repositoryGo,
	'internal/platform/es/repository.go',
	'type Projection func',
	'type RepositoryConfig[',
);
export const scenarioApi = go(
	scenarioGo,
	'internal/platform/es/estest/scenario.go',
	'func Given[',
	'func (s *Scenario[A]) When(',
	'func (s *Scenario[A]) Then(',
	'func (s *Scenario[A]) ThenError(',
);
export const storeContract = go(estestStoreGo, 'internal/platform/es/estest/store.go', 'type OpenStore', 'func TestStore(');
export const memdbContractTests = go(
	memdbTestGo,
	'internal/platform/memdb/memdb_test.go',
	'func TestEventStore(',
	'func TestOutbox(',
);
export const unitOfWork = go(uowGo, 'internal/platform/uow/uow.go', 'type Tx interface', 'type UnitOfWork interface');

export const outboxStore = go(outboxGo, 'internal/platform/outbox/outbox.go', 'type Message struct', 'type Store interface');
export const outboxNewMessage = go(outboxGo, 'internal/platform/outbox/outbox.go', 'func NewMessage(');
export const relayConfig = go(relayGo, 'internal/platform/outbox/relay.go', 'type Handler func', 'type RelayConfig struct');
// A list that pages with a cursor: the use case and the query behind it.
export const authClientsPage = go(authClientsGo, 'internal/auth/app/clients.go', 'func (s *Service) Clients(', 'func encodeClientCursor(');
export const authClientsQuery = go(authMysqlGo, 'internal/auth/store/mysql/mysql.go', 'func (s *Store) Clients(');
export const relayBackoff = go(relayGo, 'internal/platform/outbox/relay.go', 'func Backoff(');
export const relayMetrics = go(relayGo, 'internal/platform/outbox/relay.go', 'func (r *Relay) init()');
export const relayRetryTest = go(relayTestGo, 'internal/platform/outbox/relay_test.go', 'func TestRelayRetriesUntilDelivered(');
export const relayRunTest = go(relayTestGo, 'internal/platform/outbox/relay_test.go', 'func TestRelayRunPollsUntilCancelled(');

export const apperrKinds = go(
	apperrGo,
	'internal/platform/apperr/apperr.go',
	'var (',
	'func New(',
	'type ValidationError struct',
	'func Invalid(',
);

export const problemSchema: Sample = {
	code: yamlBlock(userOpenAPI, 'api/user.openapi.yaml', 'components', 'schemas', 'Problem'),
	lang: 'yaml',
	title: 'api/user.openapi.yaml',
};

export const loginService: Sample = { code: loginProto, lang: 'proto', title: 'proto/user/v1/login.proto' };
export const bufConfig: Sample = { code: bufYaml, lang: 'yaml', title: 'buf.yaml' };
export const bufGenConfig: Sample = { code: bufGenYaml, lang: 'yaml', title: 'buf.gen.yaml' };
