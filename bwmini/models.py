"""Nonsecret model intent and native-settings handoff; never an inference/credential proxy."""
import json
import re
from .domain import Fault

SCHEMA="CREATE TABLE model_selections (actor TEXT PRIMARY KEY, revision INTEGER NOT NULL, selection TEXT NOT NULL, reported_ready INTEGER NOT NULL CHECK(reported_ready IN (0,1)))"
CLIENT_VERSION='3.16.3'
CATALOG_VERSION='zcode-3.16.3/public-catalog-33'
PROVIDERS={'zai':'Z.ai','bigmodel':'BigModel'}
MODELS=('GLM-5.3','GLM-5.3-Flash')
ACCESS=('native-existing','account','api-key')

class Models:
    def __init__(self,store):self.store=store
    def actor(self,actor):
        if not isinstance(actor,str) or re.fullmatch(r'[A-Za-z0-9_-]{1,64}',actor) is None:raise Fault('INVALID_ACTOR',422)
    def revision(self,revision):
        if type(revision) is not int or not 0<=revision<2**53:raise Fault('INVALID_REVISION',422)
    def selection(self,data):
        if not isinstance(data,dict) or set(data)!={'client_version','provider','model','access_mode'}:raise Fault('INVALID_MODEL_SELECTION',422)
        if any(not isinstance(v,str) for v in data.values()):raise Fault('INVALID_MODEL_SELECTION',422)
        if data['client_version']!=CLIENT_VERSION:raise Fault('MODEL_CLIENT_UNSUPPORTED',422)
        if data['provider'] not in PROVIDERS or data['model'] not in MODELS or data['access_mode'] not in ACCESS:raise Fault('MODEL_COMBINATION_UNSUPPORTED',422)
        return dict(data)
    def catalog(self):
        return {'version':CATALOG_VERSION,'client':'zcode','client_version':CLIENT_VERSION,
                'providers':[{'id':k,'label':v} for k,v in PROVIDERS.items()], 'models':list(MODELS),'access_modes':list(ACCESS),
                'credential_handling':'native_client_only','automatic_inference':False}
    def read(self,actor='operator'):
        self.actor(actor)
        with self.store.db() as db:row=db.execute('SELECT * FROM model_selections WHERE actor=?',(actor,)).fetchone()
        return self.present(row)
    def present(self,row):
        selection=json.loads(row['selection']) if row else None
        ready=bool(row['reported_ready']) if row else False
        return {'revision':row['revision'] if row else 0,'selection':selection,
                'status':'operator_reported_ready' if ready else 'native_setup_pending' if row else 'not_selected',
                'evidence':'operator_report' if ready else 'none','inference':'unverified','catalog':self.catalog()}
    def save(self,data,revision,actor='operator'):
        self.actor(actor);self.revision(revision);selection=self.selection(data)
        encoded=json.dumps(selection,sort_keys=True)
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE')
            row=db.execute('SELECT * FROM model_selections WHERE actor=?',(actor,)).fetchone()
            if revision!=(row['revision'] if row else 0):raise Fault('MODEL_REVISION_CONFLICT',409)
            if row is None:db.execute('INSERT INTO model_selections VALUES(?,?,?,0)',(actor,1,encoded))
            elif json.loads(row['selection'])!=selection:db.execute('UPDATE model_selections SET revision=revision+1,selection=?,reported_ready=0 WHERE actor=?',(encoded,actor))
            row=db.execute('SELECT * FROM model_selections WHERE actor=?',(actor,)).fetchone()
        return self.present(row)
    def confirm(self,revision,selected,access_checked,actor='operator'):
        self.actor(actor);self.revision(revision)
        if selected is not True or access_checked is not True:raise Fault('NATIVE_CHECK_REQUIRED',422)
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE');row=db.execute('SELECT * FROM model_selections WHERE actor=?',(actor,)).fetchone()
            if row is None:raise Fault('MODEL_SELECTION_REQUIRED',409)
            if row['revision']!=revision:raise Fault('MODEL_REVISION_CONFLICT',409)
            if not row['reported_ready']:db.execute('UPDATE model_selections SET revision=revision+1,reported_ready=1 WHERE actor=?',(actor,))
            row=db.execute('SELECT * FROM model_selections WHERE actor=?',(actor,)).fetchone()
        return self.present(row)
    def handoff(self,actor='operator'):
        value=self.read(actor);selection=value['selection']
        if selection is None:raise Fault('MODEL_SELECTION_REQUIRED',409)
        return {'schema':1,'catalog_version':CATALOG_VERSION,'selection':selection,'revision':value['revision'],
                'status':value['status'],'evidence':value['evidence'],'inference':'unverified',
                'steps':['Open ZCode '+CLIENT_VERSION+' > Settings > Model Settings.',
                         'Choose '+PROVIDERS[selection['provider']]+'.',
                         'Keep the existing native ZCode connection; check its provider access.' if selection['access_mode']=='native-existing' else 'Use native account sign-in.' if selection['access_mode']=='account' else 'Enter the matching API key only in the native ZCode provider settings.',
                         'Select '+selection['model']+' in the native model selector.',
                         'Check provider access and the effective selected model in ZCode; record completion here only after you check.',
                         'A model request can consume quota. Run inference only after separately approving that test.'],
                'documentation':'https://zcode.z.ai/en/docs/configuration',
                'credentials':'not_collected','mcp_connection':'separate_check'}
