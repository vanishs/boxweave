"""Prompt composition only: this module has no provider or execution dependency."""
import json
import math
import re
import time
import uuid
from .domain import Fault

IDENT = re.compile(r"[A-Za-z0-9_-]{1,64}\Z")

def invalid():
    raise Fault("INVALID_GRAPH",422)

def string(value,limit,empty=True):
    if not isinstance(value,str) or len(value)>limit or (not empty and not value.strip()) or '\x00' in value:
        invalid()
    return value

def identifier(value):
    if not isinstance(value,str) or not IDENT.fullmatch(value):invalid()
    return value

def fields(value,keys):
    if not isinstance(value,dict) or set(value)!=set(keys):invalid()

def validate(document):
    fields(document,['name','prompt','cards','comments','links'])
    string(document['name'],80,False);string(document['prompt'],2000)
    for key in ['cards','comments','links']:
        if not isinstance(document[key],list) or len(document[key])>24:invalid()
    ids=set()
    for card in document['cards']:
        fields(card,['id','title','body','x','y','context'])
        key=identifier(card['id'])
        if key in ids:invalid()
        ids.add(key);string(card['title'],100,False);string(card['body'],2000)
        for coord in ['x','y']:
            value=card[coord]
            if type(value) not in (int,float) or not math.isfinite(value) or abs(value)>10000:invalid()
        fields(card['context'],['project','environment','tenant'])
        for value in card['context'].values():string(value,120)
    comment_ids=set()
    for comment in document['comments']:
        fields(comment,['id','text','card'])
        key=identifier(comment['id'])
        if key in comment_ids:invalid()
        comment_ids.add(key);string(comment['text'],1000,False)
        if comment['card'] is not None and (not isinstance(comment['card'],str) or comment['card'] not in ids):invalid()
    pairs=set();link_ids=set()
    for link in document['links']:
        fields(link,['id','source','target'])
        key=identifier(link['id'])
        if key in link_ids:invalid()
        link_ids.add(key)
        a,b=link['source'],link['target']
        if not isinstance(a,str) or not isinstance(b,str) or a not in ids or b not in ids or a==b:invalid()
        pair=tuple(sorted([a,b]))
        if pair in pairs:invalid()
        pairs.add(pair)
    encoded=json.dumps(document,ensure_ascii=False,allow_nan=False)
    try:size=len(encoded.encode())
    except UnicodeError:invalid()
    if size>12000:raise Fault('GRAPH_TOO_LARGE',422)
    return encoded

def revision(value):
    if type(value) is not int or value<1:raise Fault('INVALID_REVISION',400)

def safe(text):
    text=re.sub(r'-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----','[REDACTED PRIVATE KEY]',text)
    text=re.sub(r'\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b','[REDACTED TOKEN]',text)
    text=re.sub(r'''(?i)\b(api[_-]?key|token|password|secret)\s*[:=]\s*("[^"\n]*"|'[^'\n]*'|[^\s,;]+)''',lambda m:m[1]+'=[REDACTED]',text)
    return re.sub(r'([\\`*_{}\[\]()#+.!|<>:=\-])',r'\\\1',text)

def markdown(document):
    # Context labels are text, never live references or commands.
    lines=['# '+safe(document['name']),'',safe(document['prompt']),'']
    titles={c['id']:c['title'] for c in document['cards']}
    for card in document['cards']:
        lines+=['## '+safe(card['title']),'',safe(card['body']),'']
        for key,value in card['context'].items():
            if value:lines+=[key.title()+': '+safe(value)]
        lines+=['']
    if document['links']:
        lines+=['## Relations (undirected)','']
        for link in document['links']:lines+=[safe(titles[link['source']])+' ↔ '+safe(titles[link['target']])]
        lines+=['']
    if document['comments']:
        lines+=['## Comments','']
        for comment in document['comments']:
            prefix=(titles[comment['card']]+': ') if comment['card'] else ''
            lines+=[safe(prefix+comment['text']),'']
    return '\n'.join(lines).rstrip()+'\n'

class Graphs:
    def __init__(self,store):self.store=store
    def get(self,ident):
        with self.store.db() as db:
            row=db.execute('SELECT * FROM graphs WHERE id=?',(ident,)).fetchone()
        if row is None:raise Fault('NOT_FOUND',404)
        return {'id':row['id'],'revision':row['revision'],'document':json.loads(row['document'])}
    def list(self):
        with self.store.db() as db:rows=db.execute('SELECT id,revision,document FROM graphs ORDER BY rowid DESC LIMIT 100').fetchall()
        return [{'id':r['id'],'revision':r['revision'],'name':json.loads(r['document'])['name']} for r in rows]
    def save(self,document,ident=None,expected=None):
        encoded=validate(document)
        if ident is not None:revision(expected)
        key=ident or uuid.uuid4().hex
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE')
            if ident is None:
                if db.execute('SELECT count(*) FROM graphs').fetchone()[0]>=100:raise Fault('GRAPH_LIMIT',422)
                db.execute('INSERT INTO graphs VALUES(?,?,?)',(key,1,encoded))
                rev=1
            else:
                row=db.execute('SELECT revision FROM graphs WHERE id=?',(key,)).fetchone()
                if row is None:raise Fault('NOT_FOUND',404)
                if row['revision']!=expected:raise Fault('REVISION_CONFLICT')
                rev=expected+1
                db.execute('UPDATE graphs SET document=?,revision=? WHERE id=?',(encoded,rev,key))
        return {'id':key,'revision':rev,'document':json.loads(encoded)}
    def delete(self,ident,expected):
        revision(expected)
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE')
            row=db.execute('SELECT revision FROM graphs WHERE id=?',(ident,)).fetchone()
            if row is None:raise Fault('NOT_FOUND',404)
            if row['revision']!=expected:raise Fault('REVISION_CONFLICT')
            db.execute('DELETE FROM graphs WHERE id=?',(ident,))
    def copied(self,ident,expected):
        revision(expected)
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE')
            row=db.execute('SELECT revision FROM graphs WHERE id=?',(ident,)).fetchone()
            if row is None:raise Fault('NOT_FOUND',404)
            if row['revision']!=expected:raise Fault('REVISION_CONFLICT')
            db.execute('INSERT INTO graph_copies VALUES(?,?,?,?,?)',(uuid.uuid4().hex,ident,expected,time.time(),'reported_success'))
            db.execute('DELETE FROM graph_copies WHERE id IN (SELECT id FROM graph_copies ORDER BY at DESC LIMIT -1 OFFSET 1000)')
        return {'status':'reported_success','revision':expected}
    def export(self,ident):
        graph=self.get(ident)
        return {'revision':graph['revision'],'markdown':markdown(graph['document'])}
