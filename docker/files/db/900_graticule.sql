--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.15
-- Dumped by pg_dump version 9.5.22

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: joga_250k; Type: TABLE; Schema: public; Owner: osm
--

CREATE TABLE public.joga_250k (
    ogc_fid integer NOT NULL,
    wkb_geometry public.geometry(PolygonZ,4326),
    fc_nrn character varying(15),
    fc_series character varying(5),
    fc_sheet character varying(25),
    fc_classif character varying(20),
    fc_nsn character varying(15),
    fc_current character varying(3),
    fc_country character varying(50),
    fc_title character varying(200),
    fc_scale numeric(31,15),
    fc_datum character varying(3),
    fc_publish character varying(50),
    fc_utm_zon character varying(50),
    fc_contour character varying(50),
    fc_project character varying(50),
    fc_sheet_m numeric(31,15),
    fc_sheet00 numeric(31,15),
    fc_ll_long character varying(50),
    fc_xml character varying(254),
    fc_primary character varying(50),
    fc_seconda character varying(50),
    fc_third_c character varying(50),
    fc_symid character varying(50),
    fc_guid character varying(32),
    fc_spot_he numeric(31,15),
    fc_supplem character varying(50),
    fc_tint_ba numeric(31,15),
    fc_max_vo numeric(31,15),
    fc_max_v00 numeric(31,15),
    fc_max_ele character varying(20),
    fc_max_e00 character varying(20),
    fc_max_e01 character varying(20),
    fc_sheet01 numeric(31,15),
    fc_intl_bo character varying(20),
    fc_map_inf character varying(20),
    language character varying(20),
    pdf_file character varying(254),
    pdf_stored character varying(20),
    release_no character varying(254),
    id numeric(10,0),
    geometry character varying(254),
    fc_air_inf character varying(20),
    dataprovid character varying(50),
    slope_perc double precision,
    fc_elevati date,
    fc_conto00 double precision,
    elev_guide character varying(20),
    geom2d public.geometry
);


ALTER TABLE public.joga_250k OWNER TO osm;

--
-- Name: joga_250k_ogc_fid_seq; Type: SEQUENCE; Schema: public; Owner: osm
--

CREATE SEQUENCE public.joga_250k_ogc_fid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.joga_250k_ogc_fid_seq OWNER TO osm;

--
-- Name: joga_250k_ogc_fid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: osm
--

ALTER SEQUENCE public.joga_250k_ogc_fid_seq OWNED BY public.joga_250k.ogc_fid;


--
-- Name: tlm_50_index; Type: TABLE; Schema: public; Owner: osm
--

CREATE TABLE public.tlm_50_index (
    ogc_fid integer NOT NULL,
    wkb_geometry public.geometry(Polygon,4326),
    objectid numeric(9,0),
    series character varying(6),
    mdg_id character varying(20),
    nrn character varying(20),
    sheet character varying(10),
    classifica character varying(30),
    country character varying(50),
    nsn character varying(13),
    shape_leng numeric(24,15),
    guid character varying(50),
    edited character varying(50),
    created_us character varying(254),
    created_da date,
    last_edite character varying(254),
    last_edi_1 date,
    shape_ar_1 numeric(24,15),
    objectid_1 numeric(9,0),
    shape_le_1 numeric(24,15),
    item_id character varying(254),
    minlat character varying(254),
    minlon character varying(254),
    maxlat character varying(254),
    maxlon character varying(254),
    min_lat character varying(254),
    min_lon character varying(254),
    max_lat character varying(254),
    max_lon character varying(254),
    cen_lat character varying(254),
    cen_lon character varying(254),
    swlat numeric(24,15),
    swlon numeric(24,15),
    nelat numeric(24,15),
    nelon numeric(24,15),
    swlat_dms character varying(20),
    swlon_dms character varying(20),
    nelat_dms character varying(20),
    nelon_dms character varying(20),
    shape_le_2 numeric(24,15),
    shape_area numeric(24,15),
    pdf_url character varying(1024)
);


ALTER TABLE public.tlm_50_index OWNER TO osm;

--
-- Name: updated912_50k_ogc_fid_seq; Type: SEQUENCE; Schema: public; Owner: osm
--

CREATE SEQUENCE public.updated912_50k_ogc_fid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.updated912_50k_ogc_fid_seq OWNER TO osm;

