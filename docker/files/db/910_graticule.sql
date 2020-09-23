
--
-- Name: updated912_50k_ogc_fid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: osm
--

ALTER SEQUENCE public.updated912_50k_ogc_fid_seq OWNED BY public.tlm_50_index.ogc_fid;


--
-- Name: ogc_fid; Type: DEFAULT; Schema: public; Owner: osm
--

ALTER TABLE ONLY public.joga_250k ALTER COLUMN ogc_fid SET DEFAULT nextval('public.joga_250k_ogc_fid_seq'::regclass);


--
-- Name: ogc_fid; Type: DEFAULT; Schema: public; Owner: osm
--

ALTER TABLE ONLY public.tlm_50_index ALTER COLUMN ogc_fid SET DEFAULT nextval('public.updated912_50k_ogc_fid_seq'::regclass);



--
-- Name: updated912_50k_ogc_fid_seq; Type: SEQUENCE SET; Schema: public; Owner: osm
--

SELECT pg_catalog.setval('public.updated912_50k_ogc_fid_seq', 216245, true);


--
-- Name: joga_250k_pkey; Type: CONSTRAINT; Schema: public; Owner: osm
--

ALTER TABLE ONLY public.joga_250k
    ADD CONSTRAINT joga_250k_pkey PRIMARY KEY (ogc_fid);


--
-- Name: updated912_50k_pkey; Type: CONSTRAINT; Schema: public; Owner: osm
--

ALTER TABLE ONLY public.tlm_50_index
    ADD CONSTRAINT updated912_50k_pkey PRIMARY KEY (ogc_fid);


--
-- Name: idx_tlm_50_index_geometry; Type: INDEX; Schema: public; Owner: osm
--

CREATE INDEX idx_tlm_50_index_geometry ON public.tlm_50_index USING gist (wkb_geometry);


--
-- Name: joga_250k_wkb_geometry_geom_idx; Type: INDEX; Schema: public; Owner: osm
--

CREATE INDEX joga_250k_wkb_geometry_geom_idx ON public.joga_250k USING gist (wkb_geometry);


--
-- Name: updated912_50k_wkb_geometry_geom_idx; Type: INDEX; Schema: public; Owner: osm
--

CREATE INDEX updated912_50k_wkb_geometry_geom_idx ON public.tlm_50_index USING gist (wkb_geometry);


--
-- Name: TABLE joga_250k; Type: ACL; Schema: public; Owner: osm
--

REVOKE ALL ON TABLE public.joga_250k FROM PUBLIC;
REVOKE ALL ON TABLE public.joga_250k FROM osm;
GRANT ALL ON TABLE public.joga_250k TO osm;
GRANT ALL ON TABLE public.joga_250k TO graticule;


--
-- Name: TABLE tlm_50_index; Type: ACL; Schema: public; Owner: osm
--

REVOKE ALL ON TABLE public.tlm_50_index FROM PUBLIC;
REVOKE ALL ON TABLE public.tlm_50_index FROM osm;
GRANT ALL ON TABLE public.tlm_50_index TO osm;
GRANT ALL ON TABLE public.tlm_50_index TO graticule;


--
-- PostgreSQL database dump complete
--

